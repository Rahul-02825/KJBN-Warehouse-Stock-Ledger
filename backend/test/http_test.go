package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/httpapi"
)

// HTTP tests: real routes and handlers, but every request here is rejected
// before reaching the database, so no database is needed.

func TestCreateProduct_InvalidFields(t *testing.T) {
	status, body := send(t, routerWithoutDB(), "POST", "/api/products",
		`{"sku":"","name":"","unit":"box","reorder_level":"-1"}`)

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
	if errorCode(body) != "VALIDATION_ERROR" {
		t.Fatalf("code = %q, want VALIDATION_ERROR", errorCode(body))
	}
	fields := body["error"].(map[string]any)["fields"].(map[string]any)
	for _, f := range []string{"sku", "name", "unit", "reorder_level"} {
		if _, ok := fields[f]; !ok {
			t.Errorf("missing error for field %q", f)
		}
	}
}

func TestCreateProduct_BadJSON(t *testing.T) {
	bodies := []string{
		`not json`,
		`{"sku":"A","name":"B","unit":"pcs","colour":"red"}`, // unknown field
		`{"sku":42}`, // wrong type
	}
	for _, b := range bodies {
		status, body := send(t, routerWithoutDB(), "POST", "/api/products", b)
		if status != http.StatusBadRequest || errorCode(body) != "BAD_REQUEST" {
			t.Errorf("body %s: got %d %q, want 400 BAD_REQUEST", b, status, errorCode(body))
		}
	}
}

func TestGetProduct_BadID(t *testing.T) {
	for _, id := range []string{"abc", "0", "-1"} {
		status, _ := send(t, routerWithoutDB(), "GET", "/api/products/"+id, "")
		if status != http.StatusBadRequest {
			t.Errorf("id %q: status = %d, want 400", id, status)
		}
	}
}

func TestCreateLocation_CodeWithSpace(t *testing.T) {
	status, body := send(t, routerWithoutDB(), "POST", "/api/locations", `{"code":"A 01"}`)
	if status != http.StatusBadRequest || errorCode(body) != "VALIDATION_ERROR" {
		t.Fatalf("got %d %q, want 400 VALIDATION_ERROR", status, errorCode(body))
	}
}

func TestRecoverMiddleware(t *testing.T) {
	panics := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	rec := httptest.NewRecorder()
	httpapi.Recover(panics).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestInventory_BadIDs(t *testing.T) {
	paths := []string{
		"/api/inventory/abc/1",
		"/api/inventory/1/0",
		"/api/products/x/balances",
		"/api/locations/-1/balances",
	}
	for _, p := range paths {
		status, body := send(t, routerWithoutDB(), "GET", p, "")
		if status != http.StatusBadRequest || errorCode(body) != "BAD_REQUEST" {
			t.Errorf("%s: got %d %q, want 400 BAD_REQUEST", p, status, errorCode(body))
		}
	}
}

func TestMovements_RejectBadInputBeforeDatabase(t *testing.T) {
	tests := []struct {
		name, path, body, field string
	}{
		{"IN zero quantity", "/api/movements/in", `{"product_id":1,"location_id":1,"quantity":"0","unit_cost":"5","created_by":"t"}`, "quantity"},
		{"IN zero cost", "/api/movements/in", `{"product_id":1,"location_id":1,"quantity":"1","unit_cost":"0","created_by":"t"}`, "unit_cost"},
		{"IN negative cost", "/api/movements/in", `{"product_id":1,"location_id":1,"quantity":"1","unit_cost":"-5","created_by":"t"}`, "unit_cost"},
		{"IN missing created_by", "/api/movements/in", `{"product_id":1,"location_id":1,"quantity":"1","unit_cost":"5"}`, "created_by"},
		{"OUT negative quantity", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"-2","created_by":"t"}`, "quantity"},
		{"OUT missing product", "/api/movements/out", `{"location_id":1,"quantity":"2","created_by":"t"}`, "product_id"},
		{"transfer same bin", "/api/movements/transfer", `{"product_id":1,"from_location_id":1,"to_location_id":1,"quantity":"2","created_by":"t"}`, "to_location_id"},
		{"adjustment zero", "/api/movements/adjustment", `{"product_id":1,"location_id":1,"quantity":"0","reason":"damaged","created_by":"t"}`, "quantity"},
		{"adjustment bad reason", "/api/movements/adjustment", `{"product_id":1,"location_id":1,"quantity":"-1","reason":"lost","created_by":"t"}`, "reason"},
		{"adjustment other without note", "/api/movements/adjustment", `{"product_id":1,"location_id":1,"quantity":"-1","reason":"other","created_by":"t"}`, "note"},
		{"reverse without created_by", "/api/movements/1/reverse", `{}`, "created_by"},
	}
	for _, tt := range tests {
		status, body := send(t, routerWithoutDB(), "POST", tt.path, tt.body)
		if status != http.StatusBadRequest || errorCode(body) != "VALIDATION_ERROR" {
			t.Errorf("%s: got %d %q, want 400 VALIDATION_ERROR", tt.name, status, errorCode(body))
			continue
		}
		fields := body["error"].(map[string]any)["fields"].(map[string]any)
		if _, ok := fields[tt.field]; !ok {
			t.Errorf("%s: no error on %q, got %v", tt.name, tt.field, fields)
		}
	}
}

func TestMovementHistory_BadFilters(t *testing.T) {
	for _, q := range []string{"?type=SALE", "?product_id=abc", "?from=yesterday"} {
		status, _ := send(t, routerWithoutDB(), "GET", "/api/movements"+q, "")
		if status != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", q, status)
		}
	}
}

func TestProductLedger_BadID(t *testing.T) {
	status, _ := send(t, routerWithoutDB(), "GET", "/api/products/abc/ledger", "")
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", status)
	}
}

// The pool points at a port where nothing listens, so the ping fails.
func TestHealth_DatabaseDown(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	status, body := send(t, httpapi.NewRouter(pool), "GET", "/healthz", "")
	if status != http.StatusServiceUnavailable || body["status"] != "unavailable" {
		t.Fatalf("got %d %v, want 503 unavailable", status, body)
	}
}
