package test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/httpapi"
)

// Integration tests for stock movements. Each test starts from an empty database with
// product 1 = RICE-25 (pcs), product 2 = DAL (kg), location 1 = A-01, location 2 = B-01.

func setupWarehouse(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool := testDB(t)
	h := httpapi.NewRouter(pool, httpapi.Config{})
	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"RICE-25","name":"Rice 25 kg bag","unit":"pcs","reorder_level":"10"}`)
	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"DAL","name":"Toor dal","unit":"kg"}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"A-01"}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"B-01"}`)
	t.Cleanup(func() { assertLedgerMatchesCache(t, pool) })
	return h, pool
}

func receive(t *testing.T, h http.Handler, product, location int, qty, cost string) map[string]any {
	t.Helper()
	body := fmt.Sprintf(`{"product_id":%d,"location_id":%d,"quantity":"%s","unit_cost":"%s","reference":"INV-1","created_by":"test"}`,
		product, location, qty, cost)
	return mustSend(t, h, 201, "POST", "/api/movements/in", body)
}

func stockAt(t *testing.T, h http.Handler, product, location int) string {
	t.Helper()
	_, body := send(t, h, "GET", fmt.Sprintf("/api/inventory/%d/%d", product, location), "")
	return body["quantity"].(string)
}

func averageCost(t *testing.T, h http.Handler, product int) string {
	t.Helper()
	_, body := send(t, h, "GET", fmt.Sprintf("/api/products/%d", product), "")
	return body["average_cost"].(string)
}

func TestIn_UpdatesBalanceLedgerAndAverage(t *testing.T) {
	h, _ := setupWarehouse(t)

	receive(t, h, 1, 1, "50", "1000")
	m := receive(t, h, 1, 1, "50", "1200")

	if got := averageCost(t, h, 1); got != "1100" {
		t.Errorf("average = %s, want 1100", got)
	}
	if got := stockAt(t, h, 1, 1); got != "100" {
		t.Errorf("stock = %s, want 100", got)
	}
	detail := mustSend(t, h, 200, "GET", fmt.Sprintf("/api/movements/%v", m["id"]), "")
	if entries := detail["entries"].([]any); len(entries) != 1 {
		t.Errorf("IN should have 1 ledger entry, got %d", len(entries))
	}
}

func TestIn_RejectsUnknownProductLocationAndFractionalPieces(t *testing.T) {
	h, _ := setupWarehouse(t)

	status, body := send(t, h, "POST", "/api/movements/in", `{"product_id":99,"location_id":1,"quantity":"1","unit_cost":"1","created_by":"t"}`)
	if status != 404 || errorCode(body) != "NOT_FOUND" {
		t.Errorf("unknown product: %d %q", status, errorCode(body))
	}
	status, _ = send(t, h, "POST", "/api/movements/in", `{"product_id":1,"location_id":99,"quantity":"1","unit_cost":"1","created_by":"t"}`)
	if status != 404 {
		t.Errorf("unknown location: %d", status)
	}
	status, _ = send(t, h, "POST", "/api/movements/in", `{"product_id":1,"location_id":1,"quantity":"1.5","unit_cost":"1","created_by":"t"}`)
	if status != 400 {
		t.Errorf("fractional pcs: %d, want 400", status)
	}
	receive(t, h, 2, 1, "2.75", "140") // fractional kg is fine
}

func TestOut_RejectsInsufficientStock(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")

	status, body := send(t, h, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"15","created_by":"t"}`)
	if status != 409 || errorCode(body) != "INSUFFICIENT_STOCK" {
		t.Fatalf("got %d %q, want 409 INSUFFICIENT_STOCK", status, errorCode(body))
	}
	details := body["error"].(map[string]any)["details"].(map[string]any)
	if details["available"] != "10" || details["requested"] != "15" {
		t.Errorf("details = %v", details)
	}
	if got := stockAt(t, h, 1, 1); got != "10" {
		t.Errorf("stock changed to %s after a rejected OUT", got)
	}
}

func TestOut_FreezesCostAtPosting(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")

	out := mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"4","reference":"ORD-1","created_by":"t"}`)
	if out["unit_cost"] != "100" || out["total_value"] != "400" {
		t.Fatalf("OUT valued at %v / %v, want 100 / 400", out["unit_cost"], out["total_value"])
	}

	receive(t, h, 1, 1, "10", "200") // average moves to (6*100 + 10*200) / 16 = 162.5
	if got := averageCost(t, h, 1); got != "162.5" {
		t.Fatalf("average = %s, want 162.5", got)
	}
	later := mustSend(t, h, 200, "GET", fmt.Sprintf("/api/movements/%v", out["id"]), "")
	if later["unit_cost"] != "100" {
		t.Errorf("OUT cost changed to %v after a later receipt", later["unit_cost"])
	}
}

func TestOut_ConcurrentRequestsCannotGoNegative(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "5", "100")

	const attempts = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	codes := map[int]int{}
	for i := 0; i < attempts; i++ {
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

	if codes[201] != 5 || codes[409] != attempts-5 {
		t.Fatalf("status counts = %v, want 5 x 201 and %d x 409", codes, attempts-5)
	}
	if got := stockAt(t, h, 1, 1); got != "0" {
		t.Errorf("stock = %s, want 0", got)
	}
}

func TestTransfer_MovesStockAsOneMovement(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "30", "100")

	m := mustSend(t, h, 201, "POST", "/api/movements/transfer",
		`{"product_id":1,"from_location_id":1,"to_location_id":2,"quantity":"20","created_by":"t"}`)

	if stockAt(t, h, 1, 1) != "10" || stockAt(t, h, 1, 2) != "20" {
		t.Errorf("A-01 = %s, B-01 = %s, want 10 and 20", stockAt(t, h, 1, 1), stockAt(t, h, 1, 2))
	}
	if got := averageCost(t, h, 1); got != "100" {
		t.Errorf("transfer changed average to %s", got)
	}
	detail := mustSend(t, h, 200, "GET", fmt.Sprintf("/api/movements/%v", m["id"]), "")
	if entries := detail["entries"].([]any); len(entries) != 2 {
		t.Errorf("transfer should have 2 entries, got %d", len(entries))
	}
	history := mustSend(t, h, 200, "GET", "/api/movements?type=TRANSFER", "")
	if history["total"] != float64(1) {
		t.Errorf("history shows %v transfers, want 1", history["total"])
	}
}

func TestTransfer_RejectsInsufficientStock(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "30", "100")

	status, body := send(t, h, "POST", "/api/movements/transfer",
		`{"product_id":1,"from_location_id":1,"to_location_id":2,"quantity":"50","created_by":"t"}`)
	if status != 409 || errorCode(body) != "INSUFFICIENT_STOCK" {
		t.Fatalf("got %d %q", status, errorCode(body))
	}
	if stockAt(t, h, 1, 1) != "30" || stockAt(t, h, 1, 2) != "0" {
		t.Error("bins changed after a rejected transfer")
	}
}

// The destination write fails after the source was already decreased; nothing may be kept.
func TestTransfer_DestinationFailureRollsBackEverything(t *testing.T) {
	h, pool := setupWarehouse(t)
	receive(t, h, 1, 1, "30", "100")
	receive(t, h, 1, 2, "99999999", "1")

	// Push B-01 to the column's limit so adding 20 more overflows NUMERIC(18,4).
	// Ledger and cache are realigned afterwards so the drift check still passes.
	_, err := pool.Exec(context.Background(), `UPDATE stock_balances SET quantity = 99999999999990 WHERE location_id = 2`)
	if err != nil {
		t.Fatal(err)
	}

	status, _ := send(t, h, "POST", "/api/movements/transfer",
		`{"product_id":1,"from_location_id":1,"to_location_id":2,"quantity":"20","created_by":"t"}`)
	if status < 400 {
		t.Fatalf("transfer should fail, got %d", status)
	}
	if got := stockAt(t, h, 1, 1); got != "30" {
		t.Errorf("source = %s after failed transfer, want 30", got)
	}
	history := mustSend(t, h, 200, "GET", "/api/movements?type=TRANSFER", "")
	if history["total"] != float64(0) {
		t.Errorf("a failed transfer left %v movement(s)", history["total"])
	}

	_, err = pool.Exec(context.Background(), `UPDATE stock_balances SET quantity = 99999999 WHERE location_id = 2`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAdjustment(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")

	m := mustSend(t, h, 201, "POST", "/api/movements/adjustment",
		`{"product_id":1,"location_id":1,"quantity":"-3","reason":"damaged","created_by":"t"}`)
	if m["from_location_id"] != float64(1) || m["to_location_id"] != nil || m["quantity"] != "3" {
		t.Errorf("negative adjustment stored as %v", m)
	}
	if got := stockAt(t, h, 1, 1); got != "7" {
		t.Errorf("stock = %s, want 7", got)
	}

	status, body := send(t, h, "POST", "/api/movements/adjustment",
		`{"product_id":1,"location_id":1,"quantity":"-20","reason":"theft","created_by":"t"}`)
	if status != 409 || errorCode(body) != "INSUFFICIENT_STOCK" {
		t.Errorf("adjustment below zero: %d %q", status, errorCode(body))
	}

	mustSend(t, h, 201, "POST", "/api/movements/adjustment",
		`{"product_id":1,"location_id":1,"quantity":"2","reason":"other","note":"found behind rack","created_by":"t"}`)
	if got := stockAt(t, h, 1, 1); got != "9" {
		t.Errorf("stock = %s, want 9", got)
	}
	if got := averageCost(t, h, 1); got != "100" {
		t.Errorf("adjustments changed average to %s", got)
	}
}

func TestReversal(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")
	out := mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"4","created_by":"t"}`)
	outPath := fmt.Sprintf("/api/movements/%v", out["id"])

	rev := mustSend(t, h, 201, "POST", outPath+"/reverse", `{"created_by":"t","note":"wrong order"}`)
	if rev["type"] != "REVERSAL" || rev["reverses_movement_id"] != out["id"] {
		t.Fatalf("reversal = %v", rev)
	}
	if got := stockAt(t, h, 1, 1); got != "10" {
		t.Errorf("stock = %s after reversing the OUT, want 10", got)
	}

	original := mustSend(t, h, 200, "GET", outPath, "")
	if original["is_reversed"] != true || original["reversed_by_id"] != rev["id"] || original["quantity"] != "4" {
		t.Errorf("original after reversal = %v", original)
	}

	status, body := send(t, h, "POST", outPath+"/reverse", `{"created_by":"t"}`)
	if status != 409 || errorCode(body) != "ALREADY_REVERSED" {
		t.Errorf("double reversal: %d %q", status, errorCode(body))
	}

	status, body = send(t, h, "POST", fmt.Sprintf("/api/movements/%v/reverse", rev["id"]), `{"created_by":"t"}`)
	if status != 409 || errorCode(body) != "NOT_REVERSIBLE" {
		t.Errorf("reversing a reversal: %d %q", status, errorCode(body))
	}

	status, _ = send(t, h, "POST", "/api/movements/999/reverse", `{"created_by":"t"}`)
	if status != 404 {
		t.Errorf("unknown movement: %d", status)
	}
}

func TestReversal_RejectsOlderMovement(t *testing.T) {
	h, _ := setupWarehouse(t)
	in := receive(t, h, 1, 1, "10", "100")
	mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"8","created_by":"t"}`)

	status, body := send(t, h, "POST", fmt.Sprintf("/api/movements/%v/reverse", in["id"]), `{"created_by":"t"}`)
	if status != 409 || errorCode(body) != "NOT_REVERSIBLE" {
		t.Fatalf("got %d %q, want 409 NOT_REVERSIBLE", status, errorCode(body))
	}
	if got := stockAt(t, h, 1, 1); got != "2" {
		t.Errorf("stock = %s, want 2", got)
	}
}

func TestReversal_OnlyLatestMovementOfProduct(t *testing.T) {
	h, _ := setupWarehouse(t)
	first := receive(t, h, 1, 1, "100", "100")
	out := mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"100","created_by":"t"}`)
	last := receive(t, h, 1, 1, "100", "150")
	reverse := func(m map[string]any) (int, map[string]any) {
		return send(t, h, "POST", fmt.Sprintf("/api/movements/%v/reverse", m["id"]), `{"created_by":"t"}`)
	}

	if status, body := reverse(first); status != 409 || errorCode(body) != "NOT_REVERSIBLE" {
		t.Fatalf("reversing the first receipt: %d %q, want 409 NOT_REVERSIBLE", status, errorCode(body))
	}
	if got := stockAt(t, h, 1, 1); got != "100" {
		t.Errorf("stock = %s after the refused reversal, want 100", got)
	}

	for _, m := range []map[string]any{last, out, first} {
		if status, body := reverse(m); status != 201 {
			t.Fatalf("reversing movement %v: %d %q, want 201", m["id"], status, errorCode(body))
		}
	}
	if got := stockAt(t, h, 1, 1); got != "0" {
		t.Errorf("stock = %s after undoing everything, want 0", got)
	}
}

func TestReversal_OfReceiptRestoresAverage(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "50", "1000")
	second := receive(t, h, 1, 1, "50", "1200")

	mustSend(t, h, 201, "POST", fmt.Sprintf("/api/movements/%v/reverse", second["id"]), `{"created_by":"t"}`)

	if got := averageCost(t, h, 1); got != "1000" {
		t.Errorf("average = %s after reversing the second receipt, want 1000", got)
	}
	if got := stockAt(t, h, 1, 1); got != "50" {
		t.Errorf("stock = %s, want 50", got)
	}
}

func TestMovementHistoryFilters(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")
	receive(t, h, 2, 2, "5", "50")
	mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"1","reference":"ORD-77","created_by":"t"}`)

	tests := []struct {
		query string
		want  float64
	}{
		{"", 3},
		{"?product_id=1", 2},
		{"?location_id=2", 1},
		{"?type=out", 1},
		{"?reference=ORD", 1},
		{"?from=2000-01-01&to=2999-12-31", 3},
		{"?to=2000-01-01", 0},
	}
	for _, tt := range tests {
		body := mustSend(t, h, 200, "GET", "/api/movements"+tt.query, "")
		if body["total"] != tt.want {
			t.Errorf("%q: total = %v, want %v", tt.query, body["total"], tt.want)
		}
	}
}
