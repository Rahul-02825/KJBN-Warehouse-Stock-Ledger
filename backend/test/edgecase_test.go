package test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/shopspring/decimal"
)

func TestEdge_TwoOutsRacingForTheLastUnit(t *testing.T) {
	h, _ := setupWarehouse(t)
	for round := 0; round < 25; round++ {
		receive(t, h, 1, 1, "1", "100")

		var wg sync.WaitGroup
		var mu sync.Mutex
		codes := map[int]int{}
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				status, _ := send(t, h, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"1","created_by":"t"}`)
				mu.Lock()
				codes[status]++
				mu.Unlock()
			}()
		}
		wg.Wait()

		if codes[201] != 1 || codes[409] != 1 {
			t.Fatalf("round %d: codes = %v, want exactly one shipped and one refused", round, codes)
		}
		if got := stockAt(t, h, 1, 1); got != "0" {
			t.Fatalf("round %d: stock = %s, want 0", round, got)
		}
	}
}

func TestEdge_FailedTransferLeavesBothBinsUnchanged(t *testing.T) {
	h, pool := setupWarehouse(t)
	receive(t, h, 1, 1, "30", "100")
	receive(t, h, 1, 2, "99999999", "1")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE stock_balances SET quantity = 99999999999990 WHERE location_id = 2`); err != nil {
		t.Fatal(err)
	}

	status, body := send(t, h, "POST", "/api/movements/transfer", `{"product_id":1,"from_location_id":1,"to_location_id":2,"quantity":"20","created_by":"t"}`)
	if status != 400 || errorCode(body) != "OUT_OF_RANGE" {
		t.Fatalf("got %d %v, want the destination write to fail with OUT_OF_RANGE", status, body)
	}
	if stockAt(t, h, 1, 1) != "30" || stockAt(t, h, 1, 2) != "99999999999990" {
		t.Errorf("source %s, destination %s; both must be untouched", stockAt(t, h, 1, 1), stockAt(t, h, 1, 2))
	}
	var entries int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM movement_entries e JOIN movements m ON m.id = e.movement_id WHERE m.movement_type = 'TRANSFER'`).Scan(&entries); err != nil {
		t.Fatal(err)
	}
	if entries != 0 {
		t.Errorf("a failed transfer left %d ledger entries", entries)
	}
	if _, err := pool.Exec(ctx, `UPDATE stock_balances SET quantity = 99999999 WHERE location_id = 2`); err != nil {
		t.Fatal(err)
	}
}

func TestEdge_ConcurrentReversalsOfTheSameMovement(t *testing.T) {
	h, pool := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")
	out := mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"4","created_by":"t"}`)
	path := fmt.Sprintf("/api/movements/%v/reverse", out["id"])

	var wg sync.WaitGroup
	var mu sync.Mutex
	codes := map[string]int{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, body := send(t, h, "POST", path, `{"created_by":"t"}`)
			mu.Lock()
			codes[fmt.Sprintf("%d %s", status, errorCode(body))]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if codes["201 "] != 1 || codes["409 ALREADY_REVERSED"] != 9 {
		t.Fatalf("codes = %v, want one reversal and nine ALREADY_REVERSED", codes)
	}
	var reversals int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM movements WHERE reversal_of_id = $1`, out["id"]).Scan(&reversals); err != nil {
		t.Fatal(err)
	}
	if reversals != 1 || stockAt(t, h, 1, 1) != "10" {
		t.Errorf("reversals = %d, stock = %s; want 1 and 10", reversals, stockAt(t, h, 1, 1))
	}
}

func TestEdge_AverageCostStaysAccurateAfterManyReceiptsThroughTheAPI(t *testing.T) {
	h, _ := setupWarehouse(t)
	totalValue, totalQty := decimal.Zero, decimal.Zero
	for i := 1; i <= 120; i++ {
		qty := decimal.NewFromInt(int64(i%7 + 1))
		cost := decimal.NewFromInt(int64(i%13 + 1)).Div(decimal.NewFromInt(3)).Round(4)
		receive(t, h, 1, 1, qty.String(), cost.String())
		totalValue = totalValue.Add(qty.Mul(cost))
		totalQty = totalQty.Add(qty)
	}

	stored := decimal.RequireFromString(averageCost(t, h, 1))
	exact := totalValue.DivRound(totalQty, 10)
	if diff := stored.Sub(exact).Abs(); diff.GreaterThan(decimal.RequireFromString("0.001")) {
		t.Fatalf("stored average %s drifted from exact %s by %s", stored, exact, diff)
	}
	check := mustSend(t, h, 200, "GET", "/api/reports/cost-check", "")
	if check["mismatch_count"] != float64(0) {
		t.Errorf("cost check disagrees with the stored average: %v", check["data"])
	}
}

func TestEdge_DatabaseRejectsZeroCostReceiptEvenWithoutTheAPI(t *testing.T) {
	_, pool := setupWarehouse(t)
	_, err := pool.Exec(context.Background(),
		`INSERT INTO movements (product_id, movement_type, to_location_id, quantity, unit_cost, created_by) VALUES (1, 'IN', 1, 1, 0, 't')`)
	if err == nil {
		t.Fatal("the database accepted an IN with a unit cost of 0")
	}
}
