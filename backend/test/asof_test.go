package test

import (
	"context"
	"testing"
	"time"
)

func TestStockAsOf_RebuildsPastBalanceFromLedger(t *testing.T) {
	h, pool := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")
	// Pretend the first receipt happened two days ago.
	_, err := pool.Exec(context.Background(), "UPDATE movement_entries SET created_at = NOW() - INTERVAL '2 days'")
	if err != nil {
		t.Fatal(err)
	}
	mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"4","created_by":"t"}`)
	receive(t, h, 1, 2, "3", "100")

	yesterday := time.Now().In(time.FixedZone("IST", 5*3600+1800)).AddDate(0, 0, -1).Format("2006-01-02")
	body := mustSend(t, h, 200, "GET", "/api/inventory/as-of?date="+yesterday, "")
	rows := body["data"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["quantity"] != "10" {
		t.Fatalf("yesterday = %v, want only A-01 with 10", rows)
	}

	body = mustSend(t, h, 200, "GET", "/api/inventory/as-of?date="+time.Now().UTC().Add(time.Minute).Format(time.RFC3339), "")
	if rows := body["data"].([]any); len(rows) != 2 {
		t.Fatalf("now = %v, want A-01 and B-01", rows)
	}

	body = mustSend(t, h, 200, "GET", "/api/inventory/as-of?location_id=1&date="+time.Now().UTC().Add(time.Minute).Format(time.RFC3339), "")
	rows = body["data"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["quantity"] != "6" {
		t.Fatalf("A-01 now = %v, want 6", rows)
	}
}

func TestStockAsOf_RequiresValidDate(t *testing.T) {
	h := routerWithoutDB()
	for _, path := range []string{"/api/inventory/as-of", "/api/inventory/as-of?date=yesterday", "/api/inventory/as-of?date=2026-10-01&product_id=x"} {
		if status, body := send(t, h, "GET", path, ""); status != 400 || errorCode(body) != "VALIDATION_ERROR" {
			t.Errorf("%s: %d %q, want 400 VALIDATION_ERROR", path, status, errorCode(body))
		}
	}
}
