package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type ReverseRequest struct {
	Note           string `json:"note"`
	CreatedBy      string `json:"created_by"`
	IdempotencyKey string `json:"-"`
}

func (r *ReverseRequest) Validate() error {
	fields := map[string]string{}
	r.CreatedBy = checkCreatedBy(fields, r.CreatedBy)
	r.Note = checkNote(fields, r.Note)
	return result(fields)
}

func (l *Ledger) Reverse(ctx context.Context, movementID int64, req ReverseRequest) (sqlc.Movement, bool, error) {
	if err := req.Validate(); err != nil {
		return sqlc.Movement{}, false, err
	}
	movement, err := l.reverse(ctx, movementID, req)
	if errors.Is(err, errReplay) {
		return l.findReversalReplay(ctx, req.IdempotencyKey, movementID)
	}
	return movement, false, err
}

func (l *Ledger) reverse(ctx context.Context, movementID int64, req ReverseRequest) (sqlc.Movement, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return sqlc.Movement{}, err
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(tx)

	original, err := q.LockMovementForReversal(ctx, movementID)
	if postgres.IsNotFound(err) {
		return sqlc.Movement{}, domain.NotFoundError{What: "movement"}
	}
	if err != nil {
		return sqlc.Movement{}, err
	}
	if err := checkKey(ctx, q, nilIfEmpty(req.IdempotencyKey)); err != nil {
		return sqlc.Movement{}, err
	}
	if err := domain.CheckReversible(original.MovementType); err != nil {
		return sqlc.Movement{}, err
	}
	_, err = q.GetReversalForMovement(ctx, &original.ID)
	if err == nil {
		return sqlc.Movement{}, domain.ErrAlreadyReversed
	}
	if !postgres.IsNotFound(err) {
		return sqlc.Movement{}, err
	}

	product, err := q.LockProductForCostUpdate(ctx, original.ProductID)
	if err != nil {
		return sqlc.Movement{}, err
	}
	latestID, err := q.GetLatestActiveMovementID(ctx, product.ID)
	if err != nil {
		return sqlc.Movement{}, err
	}
	if latestID != original.ID {
		return sqlc.Movement{}, domain.NotReversibleError{Reason: "only the latest movement of this product can be reversed; post an adjustment instead"}
	}
	entries, err := q.GetEntriesForReversal(ctx, original.ID)
	if err != nil {
		return sqlc.Movement{}, err
	}
	changes := []stockChange{}
	for _, entry := range entries {
		changes = append(changes, stockChange{LocationID: entry.LocationID, Delta: entry.QuantityDelta.Neg()})
	}

	movement, err := saveMovement(ctx, q, product, sqlc.CreateMovementParams{
		ProductID:      product.ID,
		MovementType:   "REVERSAL",
		FromLocationID: original.ToLocationID,
		ToLocationID:   original.FromLocationID,
		Quantity:       original.Quantity,
		UnitCost:       original.UnitCost,
		Reference:      fmt.Sprintf("Reversal of movement #%d", original.ID),
		Note:           req.Note,
		CreatedBy:      req.CreatedBy,
		ReversalOfID:   &original.ID,
		IdempotencyKey: nilIfEmpty(req.IdempotencyKey),
	}, changes)
	if postgres.IsUniqueViolation(err) {
		return sqlc.Movement{}, domain.ErrAlreadyReversed
	}
	if err != nil {
		return sqlc.Movement{}, err
	}

	if original.MovementType == "IN" {
		if err := backOutReceipt(ctx, q, product, original); err != nil {
			return sqlc.Movement{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Movement{}, err
	}
	return movement, nil
}

func backOutReceipt(ctx context.Context, q *sqlc.Queries, product sqlc.Product, receipt sqlc.Movement) error {
	remaining, err := q.GetProductTotalQuantity(ctx, product.ID)
	if err != nil {
		return err
	}
	newAverage, ok := domain.BackOutAverage(remaining, product.AverageCost, receipt.Quantity, *receipt.UnitCost)
	if !ok {
		return domain.NotReversibleError{Reason: "stock valued from this receipt has already been used; post an adjustment instead"}
	}
	_, err = q.UpdateProductAverageCost(ctx, sqlc.UpdateProductAverageCostParams{ID: product.ID, AverageCost: newAverage})
	return err
}
