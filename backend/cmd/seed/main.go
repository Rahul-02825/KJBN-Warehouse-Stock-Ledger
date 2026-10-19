package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

const createdBy = "seed"

type product struct {
	sku, name, unit string
	reorderLevel    string
	unitCost        string
}

type location struct {
	code, description string
	x, y              float64
	capacity          int64
	autoPlace         bool
}

type opening struct {
	sku, location string
	quantity      string
}

var products = []product{
	{"RICE-25", "Rice 25 kg bag", "pcs", "10", "1100.00"},
	{"WHEAT-10", "Wheat flour 10 kg bag", "pcs", "20", "380.00"},
	{"SUGAR-50", "Sugar 50 kg sack", "pcs", "5", "2150.00"},
	{"DAL-TOOR", "Toor dal (loose)", "kg", "50", "142.00"},
	{"OIL-SUN", "Sunflower oil (loose)", "litre", "100", "128.50"},
	{"SALT-1", "Iodised salt 1 kg", "pcs", "40", "22.00"},
	{"TEA-500", "Tea 500 g pack", "pcs", "30", "210.00"},
	{"GHEE-1", "Ghee 1 litre tin", "litre", "15", "540.00"},
}

var warehouse = sqlc.UpsertWarehouseParams{
	Code:     "MAIN",
	Name:     "Main warehouse",
	X:        0,
	Y:        0,
	RadiusM:  50,
	Capacity: pgtype.Int8{Int64: 2000, Valid: true},
}

var locations = []location{
	{"A-01", "Rack A, bin 1", 5, 2, 150, true},
	{"A-02", "Rack A, bin 2", 5, 8, 300, true},
	{"B-01", "Rack B, bin 1", 15, 2, 300, true},
	{"B-02", "Rack B, bin 2", 15, 8, 120, true},
	{"COLD-01", "Chilled store", 30, 5, 100, true},
	{"DOCK", "Receiving dock", 0, 0, 0, false},
}

var openings = []opening{
	{"RICE-25", "A-01", "60"},
	{"RICE-25", "B-01", "25"},
	{"WHEAT-10", "A-01", "40"},
	{"WHEAT-10", "A-02", "35"},
	{"SUGAR-50", "B-02", "12"},
	{"DAL-TOOR", "A-02", "180.5"},
	{"OIL-SUN", "B-01", "240"},
	{"SALT-1", "A-02", "35"},
	{"TEA-500", "B-02", "48"},
	{"GHEE-1", "COLD-01", "22"},
}

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := run(ctx, pool); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)

	main, err := q.UpsertWarehouse(ctx, warehouse)
	if err != nil {
		return fmt.Errorf("warehouse: %w", err)
	}

	locIDs := map[string]int64{}
	newLocs := 0
	for _, l := range locations {
		row, err := q.GetLocationByCode(ctx, l.code)
		if errors.Is(err, pgx.ErrNoRows) {
			row, err = q.CreateLocation(ctx, sqlc.CreateLocationParams{Code: l.code, Description: l.description, WarehouseID: main.ID})
			newLocs++
		}
		if err != nil {
			return fmt.Errorf("location %s: %w", l.code, err)
		}
		if err := q.SetLocationLayout(ctx, sqlc.SetLocationLayoutParams{
			ID: row.ID, X: l.x, Y: l.y, Capacity: binCapacity(l.capacity), AutoPlace: l.autoPlace,
		}); err != nil {
			return fmt.Errorf("location %s layout: %w", l.code, err)
		}
		locIDs[l.code] = row.ID
	}

	// A product that already exists is assumed to be fully seeded, so its opening stock is skipped.
	prodIDs := map[string]int64{}
	fresh := map[string]bool{}
	for _, p := range products {
		row, err := q.GetProductBySKU(ctx, p.sku)
		if errors.Is(err, pgx.ErrNoRows) {
			row, err = q.CreateProduct(ctx, sqlc.CreateProductParams{
				Sku: p.sku, Name: p.name, Unit: p.unit, ReorderLevel: num(p.reorderLevel),
			})
			fresh[p.sku] = true
		}
		if err != nil {
			return fmt.Errorf("product %s: %w", p.sku, err)
		}
		prodIDs[p.sku] = row.ID
	}

	costs := map[string]string{}
	for _, p := range products {
		costs[p.sku] = p.unitCost
	}

	posted := 0
	for _, o := range openings {
		if !fresh[o.sku] {
			continue
		}
		if err := postOpening(ctx, q, prodIDs[o.sku], locIDs[o.location], o.quantity, costs[o.sku]); err != nil {
			return fmt.Errorf("opening %s@%s: %w", o.sku, o.location, err)
		}
		posted++
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("seed: %d new locations, %d new products, %d opening movements\n", newLocs, len(fresh), posted)
	return nil
}

func postOpening(ctx context.Context, q *sqlc.Queries, productID, locationID int64, qty, cost string) error {
	m, err := q.CreateMovement(ctx, sqlc.CreateMovementParams{
		ProductID:    productID,
		MovementType: "IN",
		ToLocationID: pgtype.Int8{Int64: locationID, Valid: true},
		Quantity:     num(qty),
		UnitCost:     num(cost),
		Reference:    "Opening balance",
		CreatedBy:    createdBy,
	})
	if err != nil {
		return err
	}
	if _, err := q.CreateMovementEntry(ctx, sqlc.CreateMovementEntryParams{
		MovementID: m.ID, ProductID: productID, LocationID: locationID,
		QuantityDelta: num(qty), UnitCost: num(cost),
	}); err != nil {
		return err
	}
	if err := q.EnsureStockBalanceExists(ctx, sqlc.EnsureStockBalanceExistsParams{ProductID: productID, LocationID: locationID}); err != nil {
		return err
	}
	if _, err := q.IncreaseStockBalance(ctx, sqlc.IncreaseStockBalanceParams{
		Amount: num(qty), ProductID: productID, LocationID: locationID,
	}); err != nil {
		return err
	}
	// Every opening receipt for a product uses one cost, so the average equals that cost.
	_, err = q.UpdateProductAverageCost(ctx, sqlc.UpdateProductAverageCostParams{ID: productID, AverageCost: num(cost)})
	return err
}

func binCapacity(n int64) pgtype.Int8 {
	return pgtype.Int8{Int64: n, Valid: n > 0}
}

func num(s string) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		panic(fmt.Sprintf("bad numeric literal %q: %v", s, err))
	}
	return n
}
