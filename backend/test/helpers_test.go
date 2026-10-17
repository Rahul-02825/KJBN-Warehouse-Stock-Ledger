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

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/api"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
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
	return api.NewRouter(sqlc.New(nil))
}

// routerWithDB returns the full router on an emptied test database.
func routerWithDB(t *testing.T) http.Handler {
	t.Helper()
	return api.NewRouter(sqlc.New(testDB(t)))
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

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(context.Background(),
		"TRUNCATE movement_entries, movements, stock_balances, products, locations RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatal(err)
	}
	return pool
}
