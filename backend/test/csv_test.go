package test

import (
	"encoding/csv"
	"net/http/httptest"
	"strings"
	"testing"
)

func readCSV(t *testing.T, rec *httptest.ResponseRecorder) [][]string {
	t.Helper()
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("status %d, content type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	rows, err := csv.NewReader(rec.Body).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestCSV_MovementsValuationLowStock(t *testing.T) {
	h, _ := setupWarehouse(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", `{"product_id":1,"location_id":1,"quantity":"5","unit_cost":"100","reference":"=HYPERLINK(1)","created_by":"t"}`)
	receive(t, h, 2, 1, "2", "40")

	rows := readCSV(t, sendRaw(h, "GET", "/api/movements?format=csv&product_id=1", "", nil))
	if len(rows) != 2 || rows[0][0] != "id" || rows[1][2] != "IN" {
		t.Fatalf("movements csv = %v", rows)
	}
	if rows[1][11] != "'=HYPERLINK(1)" {
		t.Errorf("reference = %q, want the formula escaped", rows[1][11])
	}

	rows = readCSV(t, sendRaw(h, "GET", "/api/reports/valuation?format=csv", "", nil))
	last := rows[len(rows)-1]
	if last[0] != "TOTAL" || last[5] != "580" {
		t.Errorf("valuation total row = %v, want TOTAL 580", last)
	}

	rows = readCSV(t, sendRaw(h, "GET", "/api/reports/low-stock?format=csv", "", nil))
	if len(rows) != 2 || rows[1][0] != "RICE-25" {
		t.Errorf("low stock csv = %v, want RICE-25 (5 <= 10)", rows)
	}
}
