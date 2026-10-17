package test

import (
	"net/http"
	"testing"
)

// Integration tests: full request through the handlers into a real PostgreSQL.
// They run only when TEST_DATABASE_URL is set.

func TestProductLifecycle(t *testing.T) {
	h := routerWithDB(t)

	status, body := send(t, h, "POST", "/api/products",
		`{"sku":"RICE-25","name":"Rice 25 kg bag","unit":"pcs","reorder_level":"10"}`)
	if status != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %v", status, body)
	}
	if body["sku"] != "RICE-25" || body["reorder_level"] != "10" || body["average_cost"] != "0" {
		t.Fatalf("create: unexpected body %v", body)
	}

	status, body = send(t, h, "POST", "/api/products",
		`{"sku":"RICE-25","name":"Duplicate","unit":"pcs"}`)
	if status != http.StatusConflict || errorCode(body) != "DUPLICATE" {
		t.Fatalf("duplicate: got %d %q, want 409 DUPLICATE", status, errorCode(body))
	}

	status, body = send(t, h, "GET", "/api/products/1", "")
	if status != http.StatusOK || body["name"] != "Rice 25 kg bag" {
		t.Fatalf("get: got %d %v", status, body)
	}

	status, body = send(t, h, "PATCH", "/api/products/1", `{"reorder_level":"15"}`)
	if status != http.StatusOK || body["reorder_level"] != "15" || body["name"] != "Rice 25 kg bag" {
		t.Fatalf("patch: got %d %v", status, body)
	}

	status, body = send(t, h, "GET", "/api/products?search=rice", "")
	if status != http.StatusOK || body["total"] != float64(1) {
		t.Fatalf("list: got %d %v", status, body)
	}

	status, _ = send(t, h, "GET", "/api/products/999", "")
	if status != http.StatusNotFound {
		t.Fatalf("missing: status = %d, want 404", status)
	}
}

func TestLocationLifecycle(t *testing.T) {
	h := routerWithDB(t)

	status, body := send(t, h, "POST", "/api/locations", `{"code":"A-01","description":"Rack A, bin 1"}`)
	if status != http.StatusCreated || body["code"] != "A-01" {
		t.Fatalf("create: got %d %v", status, body)
	}

	status, body = send(t, h, "POST", "/api/locations", `{"code":"A-01"}`)
	if status != http.StatusConflict || errorCode(body) != "DUPLICATE" {
		t.Fatalf("duplicate: got %d %q, want 409 DUPLICATE", status, errorCode(body))
	}

	status, body = send(t, h, "GET", "/api/locations", "")
	if status != http.StatusOK || len(body["data"].([]any)) != 1 {
		t.Fatalf("list: got %d %v", status, body)
	}

	status, _ = send(t, h, "GET", "/api/locations/999", "")
	if status != http.StatusNotFound {
		t.Fatalf("missing: status = %d, want 404", status)
	}
}
