package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/api"
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
	api.Recover(panics).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
