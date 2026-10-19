package test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/httpapi"
)

func setupLayout(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool := testDB(t)
	h := httpapi.NewRouter(pool)
	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"RICE","name":"Rice bag","unit":"pcs"}`)
	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"WHEAT","name":"Wheat bag","unit":"pcs"}`)
	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"DAL","name":"Toor dal","unit":"kg"}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"A-01","x":5,"y":2,"capacity":60}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"A-02","x":5,"y":8,"capacity":60}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"B-01","x":15,"y":2,"capacity":100}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"DOCK","x":0,"y":0,"auto_place":false}`)
	t.Cleanup(func() { assertLedgerMatchesCache(t, pool) })
	return h, pool
}

func receiveInto(product int, location string, qty int) string {
	bin := ""
	if location != "" {
		bin = fmt.Sprintf(`"location_id":%s,`, location)
	}
	return fmt.Sprintf(`{"product_id":%d,%s"quantity":"%d","unit_cost":"10","created_by":"t"}`, product, bin, qty)
}

func entriesByBin(t *testing.T, h http.Handler, movementID any) map[float64]string {
	t.Helper()
	detail := mustSend(t, h, 200, "GET", fmt.Sprintf("/api/movements/%v", movementID), "")
	out := map[float64]string{}
	for _, e := range detail["entries"].([]any) {
		entry := e.(map[string]any)
		out[entry["location_id"].(float64)] = entry["quantity_delta"].(string)
	}
	return out
}

func countMovements(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM movements").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestBinLayout_Validation(t *testing.T) {
	pool := testDB(t)
	h := httpapi.NewRouter(pool)
	if _, err := pool.Exec(context.Background(), "UPDATE warehouses SET radius_m = 20, capacity = 100"); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		`{"code":"FAR","x":30,"y":0}`:        "x",
		`{"code":"ZERO","capacity":0}`:       "capacity",
		`{"code":"HUGE","capacity":150}`:     "capacity",
		`{"code":"NEG","x":-15,"y":-15}`:     "x",
		`{"code":"BADWH","warehouse_id":-1}`: "warehouse_id",
	}
	for body, field := range cases {
		status, resp := send(t, h, "POST", "/api/locations", body)
		fields, _ := resp["error"].(map[string]any)["fields"].(map[string]any)
		if status != 400 || fields[field] == nil {
			t.Errorf("%s: %d %v, want 400 on %s", body, status, resp, field)
		}
	}

	bin := mustSend(t, h, 201, "POST", "/api/locations", `{"code":"OK","x":12,"y":-9,"capacity":100}`)
	if bin["capacity"] != float64(100) || bin["x"] != float64(12) || bin["auto_place"] != true {
		t.Errorf("created bin = %v", bin)
	}
	if status, _ := send(t, h, "POST", "/api/locations", `{"code":"NOWH","warehouse_id":99}`); status != 404 {
		t.Errorf("unknown warehouse: %d, want 404", status)
	}
}

func TestReceive_SplitsAcrossNearestBins(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 45))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(2, "2", 20))

	m := mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 60))
	if m["to_location_id"] != float64(1) {
		t.Errorf("to_location_id = %v, want the chosen bin A-01", m["to_location_id"])
	}
	got := entriesByBin(t, h, m["id"])
	if got[1] != "15" || got[2] != "40" || got[3] != "5" || len(got) != 3 {
		t.Fatalf("entries = %v, want A-01 15, A-02 40, B-01 5", got)
	}
	if stockAt(t, h, 1, 1) != "60" || stockAt(t, h, 1, 2) != "40" || stockAt(t, h, 2, 2) != "20" {
		t.Error("balances do not match the split")
	}
}

func TestReceive_DefaultStartBin(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "3", 10))

	m := mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "", 5))
	if m["to_location_id"] != float64(3) {
		t.Errorf("RICE already in B-01, start = %v, want 3", m["to_location_id"])
	}
	m = mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(2, "", 5))
	if m["to_location_id"] != float64(1) {
		t.Errorf("new product start = %v, want A-01 nearest the entrance (DOCK is manual only)", m["to_location_id"])
	}
}

func TestReceive_RejectsWhenNothingFits(t *testing.T) {
	h, pool := setupLayout(t)
	before := countMovements(t, pool)

	status, body := send(t, h, "POST", "/api/movements/in", receiveInto(1, "1", 221))
	if status != 409 || errorCode(body) != "NO_SPACE" {
		t.Fatalf("got %d %v, want 409 NO_SPACE", status, body)
	}
	details := body["error"].(map[string]any)["details"].(map[string]any)
	if details["fit"] != "220" || details["requested"] != "221" {
		t.Errorf("details = %v, want fit 220 of 221", details)
	}
	if countMovements(t, pool) != before {
		t.Error("a rejected receipt must not post anything")
	}
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 220))
}

func TestReceive_ChosenManualBinTakesStock(t *testing.T) {
	h, _ := setupLayout(t)
	m := mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "4", 500))
	if got := entriesByBin(t, h, m["id"]); got[4] != "500" || len(got) != 1 {
		t.Errorf("entries = %v, want everything in the unlimited DOCK", got)
	}
}

func TestReceive_WarehouseCapacityLimitsUnlimitedBins(t *testing.T) {
	h, pool := setupLayout(t)
	if _, err := pool.Exec(context.Background(), "UPDATE warehouses SET capacity = 50"); err != nil {
		t.Fatal(err)
	}
	status, body := send(t, h, "POST", "/api/movements/in", receiveInto(1, "4", 51))
	if status != 409 || body["error"].(map[string]any)["details"].(map[string]any)["fit"] != "50" {
		t.Fatalf("got %d %v, want NO_SPACE with 50 fitting", status, body)
	}
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "4", 50))
}

func TestCapacity_OtherMovementsIntoFullBin(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 60))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "3", 10))

	status, body := send(t, h, "POST", "/api/movements/transfer", `{"product_id":1,"from_location_id":3,"to_location_id":1,"quantity":"1","created_by":"t"}`)
	if status != 409 || errorCode(body) != "NO_SPACE" {
		t.Errorf("transfer into full bin: %d %v", status, body)
	}
	status, body = send(t, h, "POST", "/api/movements/adjustment", `{"product_id":1,"location_id":1,"quantity":"1","reason":"count_correction","created_by":"t"}`)
	if status != 409 || errorCode(body) != "NO_SPACE" {
		t.Errorf("adjustment into full bin: %d %v", status, body)
	}

	out := mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"5","created_by":"t"}`)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(2, "1", 5))
	status, body = send(t, h, "POST", fmt.Sprintf("/api/movements/%v/reverse", out["id"]), `{"created_by":"t"}`)
	if status != 409 || errorCode(body) != "NO_SPACE" {
		t.Errorf("reversal into full bin: %d %v", status, body)
	}
}

func receiveKg(location string, qty string) string {
	return fmt.Sprintf(`{"product_id":3,"location_id":%s,"quantity":"%s","unit_cost":"10","created_by":"t"}`, location, qty)
}

func TestCapacity_KgSplitsAcrossBinsInFractions(t *testing.T) {
	h, _ := setupLayout(t)
	m := mustSend(t, h, 201, "POST", "/api/movements/in", receiveKg("1", "70.25"))
	if got := entriesByBin(t, h, m["id"]); got[1] != "60" || got[2] != "10.25" || len(got) != 2 {
		t.Fatalf("entries = %v, want A-01 60 and A-02 10.25", got)
	}
	m = mustSend(t, h, 201, "POST", "/api/movements/in", `{"product_id":3,"quantity":"5","unit_cost":"10","created_by":"t"}`)
	if m["to_location_id"] != float64(2) {
		t.Errorf("kg without a bin should start where the most dal is with space (A-02), got %v", m["to_location_id"])
	}
}

func TestCapacity_KgCountsAndPiecesStayWhole(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveKg("1", "50.5"))
	m := mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 10))
	if got := entriesByBin(t, h, m["id"]); got[1] != "9" || got[2] != "1" {
		t.Fatalf("entries = %v, want 9 whole pieces in A-01 (9.5 units free) and 1 in A-02", got)
	}
}

func TestReceive_ConcurrentReceiptsNeverOverfillABin(t *testing.T) {
	pool := testDB(t)
	h := httpapi.NewRouter(pool)
	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"RICE","name":"Rice","unit":"pcs"}`)
	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"WHEAT","name":"Wheat","unit":"pcs"}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"ONLY","capacity":10}`)
	t.Cleanup(func() { assertLedgerMatchesCache(t, pool) })

	var mu sync.Mutex
	var wg sync.WaitGroup
	codes := map[int]int{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(product int) {
			defer wg.Done()
			status, _ := send(t, h, "POST", "/api/movements/in", receiveInto(product, "1", 1))
			mu.Lock()
			codes[status]++
			mu.Unlock()
		}(i%2 + 1)
	}
	wg.Wait()

	if codes[201] != 10 || codes[409] != 10 {
		t.Fatalf("codes = %v, want 10 accepted and 10 rejected", codes)
	}
	locations := mustSend(t, h, 200, "GET", "/api/locations", "")
	if used := locations["data"].([]any)[0].(map[string]any)["used"]; used != "10" {
		t.Errorf("used = %v, want exactly the capacity of 10", used)
	}
}

func TestPreview_ShowsPlanWithoutPosting(t *testing.T) {
	h, pool := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 45))
	before := countMovements(t, pool)

	body := mustSend(t, h, 200, "POST", "/api/placements/preview", `{"product_id":1,"quantity":"30"}`)
	if body["start_location_code"] != "A-01" || body["reason"] != domain.ReasonProductBin {
		t.Errorf("start = %v (%v)", body["start_location_code"], body["reason"])
	}
	allocations := body["allocations"].([]any)
	second := allocations[1].(map[string]any)
	if len(allocations) != 2 || second["location_code"] != "A-02" || second["quantity"] != "15" || second["distance_m"] != float64(6) {
		t.Errorf("allocations = %v", allocations)
	}
	if countMovements(t, pool) != before {
		t.Error("preview must not post")
	}

	if status, body := send(t, h, "POST", "/api/placements/preview", `{"product_id":1,"quantity":"1000"}`); status != 409 || errorCode(body) != "NO_SPACE" {
		t.Errorf("too much: %d %v", status, body)
	}
	kg := mustSend(t, h, 200, "POST", "/api/placements/preview", `{"product_id":3,"quantity":"2.5"}`)
	if kg["unit"] != "kg" || kg["allocations"].([]any)[0].(map[string]any)["quantity"] != "2.5" {
		t.Errorf("kg preview = %v", kg)
	}
	if status, _ := send(t, h, "POST", "/api/placements/preview", `{"product_id":1,"quantity":"1.5"}`); status != 400 {
		t.Errorf("fractional pcs preview: %d, want 400", status)
	}
}

func TestPreview_RejectsBadInput(t *testing.T) {
	h := routerWithoutDB()
	for _, body := range []string{`{}`, `{"product_id":1,"quantity":"0"}`, `{"product_id":1,"quantity":"2","location_id":-1}`} {
		if status, resp := send(t, h, "POST", "/api/placements/preview", body); status != 400 {
			t.Errorf("%s: %d %v, want 400", body, status, resp)
		}
	}
}
