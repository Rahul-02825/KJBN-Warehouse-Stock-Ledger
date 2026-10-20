package ledger

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

func saveMovement(ctx context.Context, q *sqlc.Queries, product sqlc.Product, row sqlc.CreateMovementParams, changes []stockChange) (sqlc.Movement, error) {
	if err := checkKey(ctx, q, row.IdempotencyKey); err != nil {
		return sqlc.Movement{}, err
	}
	if err := ensureSpace(ctx, q, product, changes); err != nil {
		return sqlc.Movement{}, err
	}

	for _, change := range changes {
		var err error
		if change.Delta.IsPositive() {
			err = addStock(ctx, q, product.ID, change.LocationID, change.Delta)
		} else {
			err = takeStock(ctx, q, product.ID, change.LocationID, change.Delta.Neg())
		}
		if err != nil {
			return sqlc.Movement{}, err
		}
	}

	movement, err := q.CreateMovement(ctx, row)
	if postgres.ConstraintName(err) == "movements_idempotency_key_key" {
		return sqlc.Movement{}, errReplay
	}
	if err != nil {
		return sqlc.Movement{}, err
	}

	for _, change := range changes {
		_, err := q.CreateMovementEntry(ctx, sqlc.CreateMovementEntryParams{
			MovementID:    movement.ID,
			ProductID:     movement.ProductID,
			LocationID:    change.LocationID,
			QuantityDelta: change.Delta,
			UnitCost:      row.UnitCost,
		})
		if err != nil {
			return sqlc.Movement{}, err
		}
	}
	return movement, nil
}

func addStock(ctx context.Context, q *sqlc.Queries, productID, locationID int64, quantity decimal.Decimal) error {
	err := q.EnsureStockBalanceExists(ctx, sqlc.EnsureStockBalanceExistsParams{ProductID: productID, LocationID: locationID})
	if err != nil {
		return err
	}
	_, err = q.IncreaseStockBalance(ctx, sqlc.IncreaseStockBalanceParams{
		Amount:     quantity,
		ProductID:  productID,
		LocationID: locationID,
	})
	return err
}

func takeStock(ctx context.Context, q *sqlc.Queries, productID, locationID int64, quantity decimal.Decimal) error {
	updated, err := q.DecreaseStockBalanceIfAvailable(ctx, sqlc.DecreaseStockBalanceIfAvailableParams{
		Amount:     quantity,
		ProductID:  productID,
		LocationID: locationID,
	})
	if err != nil {
		return err
	}
	if updated == 1 {
		return nil
	}

	balance, err := q.GetStockBalance(ctx, sqlc.GetStockBalanceParams{ProductID: productID, LocationID: locationID})
	if err != nil {
		return err
	}
	return domain.InsufficientStockError{Location: balance.LocationCode, Available: balance.Quantity, Requested: quantity}
}

func lockProduct(ctx context.Context, q *sqlc.Queries, productID int64, quantity decimal.Decimal) (sqlc.Product, error) {
	product, err := q.LockProductForCostUpdate(ctx, productID)
	if postgres.IsNotFound(err) {
		return sqlc.Product{}, domain.NotFoundError{What: "product"}
	}
	if err != nil {
		return sqlc.Product{}, err
	}
	if msg := domain.CheckWholeQuantity(quantity, product.Unit); msg != "" {
		return sqlc.Product{}, domain.ValidationError{"quantity": msg}
	}
	return product, nil
}

func getLocation(ctx context.Context, q *sqlc.Queries, id int64) (sqlc.Location, error) {
	location, err := q.GetLocationByID(ctx, id)
	if postgres.IsNotFound(err) {
		return sqlc.Location{}, domain.NotFoundError{What: "location"}
	}
	return location, err
}

func checkKey(ctx context.Context, q *sqlc.Queries, key *string) error {
	if key == nil {
		return nil
	}
	_, err := q.GetMovementByIdempotencyKey(ctx, key)
	if err == nil {
		return errReplay
	}
	if postgres.IsNotFound(err) {
		return nil
	}
	return err
}
