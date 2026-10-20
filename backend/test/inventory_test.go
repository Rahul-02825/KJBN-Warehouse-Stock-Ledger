package test

import (
	"net/http"
	"testing"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/httpapi"
)

// Integration test for the read-only inventory endpoints.
func TestInventoryBalances(t *testing.T) {
	h := httpapi.NewRouter(testDB(t), httpapi.Config{})

	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"RICE-25","name":"Rice","unit":"pcs"}`)
	mustSend(t, h, 201, "POST", "/api/products", `{"sku":"DAL","name":"Dal","unit":"kg"}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"A-01"}`)
	mustSend(t, h, 201, "POST", "/api/locations", `{"code":"B-01"}`)
	for _, body := range []string{
		`{"product_id":1,"location_id":1,"quantity":"30","unit_cost":"10","created_by":"test"}`,
		`{"product_id":1,"location_id":2,"quantity":"20","unit_cost":"10","created_by":"test"}`,
		`{"product_id":2,"location_id":1,"quantity":"12.5","unit_cost":"4","created_by":"test"}`,
	} {
		mustSend(t, h, 201, "POST", "/api/movements/in", body)
	}

	t.Run("grid has every product x location", func(t *testing.T) {
		status, body := send(t, h, "GET", "/api/inventory", "")
		cells := body["data"].([]any)
		if status != http.StatusOK || len(cells) != 4 {
			t.Fatalf("got %d with %d cells, want 200 with 4", status, len(cells))
		}
	})

	t.Run("single cell, including an empty one", func(t *testing.T) {
		_, body := send(t, h, "GET", "/api/inventory/1/2", "")
		if body["quantity"] != "20" {
			t.Fatalf("RICE-25 @ B-01 = %v, want 20", body["quantity"])
		}
		_, body = send(t, h, "GET", "/api/inventory/2/2", "")
		if body["quantity"] != "0" {
			t.Fatalf("DAL @ B-01 = %v, want 0", body["quantity"])
		}
		status, _ := send(t, h, "GET", "/api/inventory/1/99", "")
		if status != http.StatusNotFound {
			t.Fatalf("unknown location: status %d, want 404", status)
		}
	})

	t.Run("balances by product with total", func(t *testing.T) {
		status, body := send(t, h, "GET", "/api/products/1/balances", "")
		if status != http.StatusOK || body["total_quantity"] != "50" || len(body["data"].([]any)) != 2 {
			t.Fatalf("got %d %v", status, body)
		}
		status, _ = send(t, h, "GET", "/api/products/99/balances", "")
		if status != http.StatusNotFound {
			t.Fatalf("unknown product: status %d, want 404", status)
		}
	})

	t.Run("balances by location", func(t *testing.T) {
		status, body := send(t, h, "GET", "/api/locations/1/balances", "")
		if status != http.StatusOK || body["location_code"] != "A-01" || len(body["data"].([]any)) != 2 {
			t.Fatalf("got %d %v", status, body)
		}
	})
}
