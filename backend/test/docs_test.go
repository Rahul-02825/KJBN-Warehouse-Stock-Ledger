package test

import (
	"strings"
	"testing"
)

func TestDocs_ServesSpecAndSwaggerUI(t *testing.T) {
	h := routerWithoutDB()
	rec := sendRaw(h, "GET", "/api/openapi.yaml", "", nil)
	if rec.Code != 200 || !strings.HasPrefix(rec.Body.String(), "openapi: 3.0.3") {
		t.Fatalf("spec: %d", rec.Code)
	}
	rec = sendRaw(h, "GET", "/api/docs", "", nil)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "/api/openapi.yaml") {
		t.Fatalf("docs page: %d", rec.Code)
	}
}
