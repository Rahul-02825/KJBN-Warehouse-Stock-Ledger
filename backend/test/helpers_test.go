package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/httpapi"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
)

// send makes a request to the handler and returns the status code and decoded JSON body.
func send(t *testing.T, h http.Handler, method, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// errorCode reads error.code from an error response.
func errorCode(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

// routerWithoutDB is enough for requests that are rejected before any query runs.
func routerWithoutDB() http.Handler {
	return httpapi.NewRouter(nil)
}

// routerWithDB returns the full router on an emptied test database.
func routerWithDB(t *testing.T) http.Handler {
	t.Helper()
	return httpapi.NewRouter(testDB(t))
}

// testDB connects to TEST_DATABASE_URL and empties all tables first.
// The test is skipped when the variable is not set.
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	if strings.Contains(url, "supabase") {
		t.Fatal("TEST_DATABASE_URL points at Supabase; tests delete all data, use a local database")
	}

	pool, err := postgres.Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(context.Background(),
		"TRUNCATE movement_entries, movements, stock_balances, products, locations RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(),
		"UPDATE warehouses SET x = 0, y = 0, radius_m = 1000, capacity = NULL WHERE code = 'MAIN'")
	if err != nil {
		t.Fatal(err)
	}
	return pool
}

// mustSend fails the test unless the response has the wanted status.
func mustSend(t *testing.T, h http.Handler, want int, method, path, body string) map[string]any {
	t.Helper()
	status, out := send(t, h, method, path, body)
	if status != want {
		t.Fatalf("%s %s: status %d, want %d, body %v", method, path, status, want, out)
	}
	return out
}

// assertLedgerMatchesCache checks that every cached balance equals the sum of its ledger entries.
func assertLedgerMatchesCache(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var drift int
	err := pool.QueryRow(context.Background(), `
		WITH lt AS (SELECT product_id, location_id, SUM(quantity_delta) q FROM movement_entries GROUP BY 1, 2)
		SELECT count(*) FROM stock_balances sb FULL JOIN lt USING (product_id, location_id)
		WHERE COALESCE(sb.quantity, 0) <> COALESCE(lt.q, 0)`).Scan(&drift)
	if err != nil {
		t.Fatal(err)
	}
	if drift != 0 {
		t.Fatalf("%d balances differ from the ledger", drift)
	}
}

// sendRaw makes a request with optional headers and returns the raw recorder.
func sendRaw(h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func withKey(key string) map[string]string { return map[string]string{"Idempotency-Key": key} }
