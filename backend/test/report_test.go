package test

import (
	"context"
	"testing"
)

// Integration tests for reports. setupWarehouse gives RICE-25 (pcs, reorder 10),
// DAL (kg, reorder 0), bins A-01 and B-01.

func TestLowStock_AtOrBelowReorderLevel(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "12", "100")
	receive(t, h, 2, 1, "5", "40")

	body := mustSend(t, h, 200, "GET", "/api/reports/low-stock", "")
	if n := len(body["data"].([]any)); n != 0 {
		t.Fatalf("12 > 10 and 5 > 0, so nothing is low; got %d rows", n)
	}

	// Ship 2, leaving exactly the reorder level of 10: "at or below" must include it.
	mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"2","created_by":"t"}`)
	body = mustSend(t, h, 200, "GET", "/api/reports/low-stock", "")
	rows := body["data"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["sku"] != "RICE-25" || rows[0].(map[string]any)["total_quantity"] != "10" {
		t.Fatalf("low stock = %v, want only RICE-25 at 10", rows)
	}
}

func TestValuation(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100") // 1000
	receive(t, h, 2, 2, "2.5", "40") // 100

	body := mustSend(t, h, 200, "GET", "/api/reports/valuation", "")
	if body["total_value"] != "1100" {
		t.Fatalf("total value = %v, want 1100", body["total_value"])
	}
	if n := len(body["data"].([]any)); n != 2 {
		t.Fatalf("got %d products, want 2", n)
	}
}

func TestReconcile_DetectsCorruptedCache(t *testing.T) {
	h, pool := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")

	body := mustSend(t, h, 200, "GET", "/api/reports/reconciliation", "")
	if body["drift_count"] != float64(0) {
		t.Fatalf("clean ledger reports drift: %v", body)
	}

	// Corrupt the cache: the floor "says" 7 but the ledger says 10.
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE stock_balances SET quantity = 7 WHERE product_id = 1 AND location_id = 1`); err != nil {
		t.Fatal(err)
	}

	body = mustSend(t, h, 200, "GET", "/api/reports/reconciliation?only_drift=true", "")
	rows := body["data"].([]any)
	if body["drift_count"] != float64(1) || len(rows) != 1 {
		t.Fatalf("want exactly 1 drifted cell, got %v", body)
	}
	row := rows[0].(map[string]any)
	if row["location_code"] != "A-01" || row["cached_quantity"] != "7" || row["ledger_quantity"] != "10" || row["difference"] != "-3" {
		t.Fatalf("drift row = %v", row)
	}

	// Put the cache back so the end-of-test ledger check passes.
	if _, err := pool.Exec(ctx, `UPDATE stock_balances SET quantity = 10 WHERE product_id = 1 AND location_id = 1`); err != nil {
		t.Fatal(err)
	}
}

func TestProductLedger_RunningBalances(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")
	mustSend(t, h, 201, "POST", "/api/movements/transfer", `{"product_id":1,"from_location_id":1,"to_location_id":2,"quantity":"4","created_by":"t"}`)
	mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":2,"quantity":"1","created_by":"t"}`)

	body := mustSend(t, h, 200, "GET", "/api/products/1/ledger", "")
	rows := body["data"].([]any)

	// IN +10 A-01, TRANSFER -4 A-01 and +4 B-01, OUT -1 B-01
	wantTotals := []string{"10", "6", "10", "9"}
	if len(rows) != len(wantTotals) {
		t.Fatalf("got %d ledger rows, want %d", len(rows), len(wantTotals))
	}
	for i, want := range wantTotals {
		if got := rows[i].(map[string]any)["running_total"]; got != want {
			t.Errorf("row %d running_total = %v, want %s", i, got, want)
		}
	}
	last := rows[3].(map[string]any)
	if last["location_code"] != "B-01" || last["running_location_balance"] != "3" {
		t.Errorf("last row = %v, want B-01 with running balance 3", last)
	}

	status, _ := send(t, h, "GET", "/api/products/99/ledger", "")
	if status != 404 {
		t.Errorf("unknown product: %d, want 404", status)
	}
}
