package test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/httpapi"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func logLines(t *testing.T, buf *bytes.Buffer, msg string) []map[string]any {
	t.Helper()
	var found []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	for scanner.Scan() {
		var line map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			t.Fatalf("log line is not JSON: %s", scanner.Text())
		}
		if line["msg"] == msg {
			found = append(found, line)
		}
	}
	return found
}

func TestRequestID_IsReturnedOnEveryResponse(t *testing.T) {
	h := routerWithoutDB()

	generated := sendRaw(h, "GET", "/api/products/abc", "", nil).Header().Get("X-Request-ID")
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(generated) {
		t.Errorf("generated id = %q, want 16 hex characters", generated)
	}

	echoed := sendRaw(h, "GET", "/api/products/abc", "", map[string]string{"X-Request-ID": "client-42"}).Header().Get("X-Request-ID")
	if echoed != "client-42" {
		t.Errorf("a safe client id should be kept, got %q", echoed)
	}

	replaced := sendRaw(h, "GET", "/api/products/abc", "", map[string]string{"X-Request-ID": "bad id {}"}).Header().Get("X-Request-ID")
	if replaced == "bad id {}" || replaced == "" {
		t.Errorf("an unsafe client id should be replaced, got %q", replaced)
	}
}

func TestRejectedShipment_LogsRequestAndDomainIDs(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "5", "100")
	logs := captureLogs(t)

	rec := sendRaw(h, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"9","created_by":"t"}`, nil)
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409", rec.Code)
	}

	rejected := logLines(t, logs, "request rejected")
	if len(rejected) != 1 {
		t.Fatalf("want 1 rejected line, got %d: %s", len(rejected), logs.String())
	}
	line := rejected[0]
	if line["level"] != "WARN" || line["request_id"] != rec.Header().Get("X-Request-ID") ||
		line["product_id"] != float64(1) || line["location_id"] != float64(1) || line["quantity"] != "9" {
		t.Errorf("rejected line = %v", line)
	}

	requests := logLines(t, logs, "request")
	if len(requests) != 1 || requests[0]["request_id"] != line["request_id"] {
		t.Errorf("the request line should carry the same id: %v", requests)
	}
}

func TestReversal_LogsMovementID(t *testing.T) {
	h, _ := setupWarehouse(t)
	logs := captureLogs(t)

	send(t, h, "POST", "/api/movements/999/reverse", `{"created_by":"t"}`)

	rejected := logLines(t, logs, "request rejected")
	if len(rejected) != 1 || rejected[0]["movement_id"] != float64(999) {
		t.Errorf("rejected reversal should log movement_id 999: %v", rejected)
	}
}

func unreachableDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestInternalError_ProductionLogsHideTheErrorText(t *testing.T) {
	for _, production := range []bool{false, true} {
		logs := captureLogs(t)
		h := httpapi.NewRouter(unreachableDB(t), httpapi.Config{Production: production})

		status, body := send(t, h, "GET", "/api/products", "")
		if status != 500 || errorCode(body) != "INTERNAL" {
			t.Fatalf("production=%v: got %d %v, want 500 INTERNAL", production, status, body)
		}
		lines := logLines(t, logs, "internal error")
		if len(lines) != 1 {
			t.Fatalf("production=%v: want 1 internal error line, got %s", production, logs.String())
		}
		_, hasReason := lines[0]["reason"]
		if production && (hasReason || lines[0]["error_kind"] == nil || strings.Contains(logs.String(), "127.0.0.1")) {
			t.Errorf("production log should hide the error text: %v", lines[0])
		}
		if !production && !hasReason {
			t.Errorf("development log should keep the error text: %v", lines[0])
		}
	}
}

func TestCORS_AllowsOnlyTheConfiguredOrigin(t *testing.T) {
	h := httpapi.NewRouter(nil, httpapi.Config{AllowedOrigins: []string{"https://ledger.vercel.app"}})

	allowed := sendRaw(h, "GET", "/api/products/abc", "", map[string]string{"Origin": "https://ledger.vercel.app"})
	if allowed.Header().Get("Access-Control-Allow-Origin") != "https://ledger.vercel.app" ||
		!strings.Contains(allowed.Header().Get("Access-Control-Expose-Headers"), "X-Request-ID") {
		t.Errorf("allowed origin headers = %v", allowed.Header())
	}

	other := sendRaw(h, "GET", "/api/products/abc", "", map[string]string{"Origin": "https://evil.example"})
	if other.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("an unknown origin must not be allowed: %v", other.Header())
	}

	preflight := sendRaw(h, "OPTIONS", "/api/movements/out", "", map[string]string{
		"Origin": "https://ledger.vercel.app", "Access-Control-Request-Method": "POST",
	})
	if preflight.Code != 204 || !strings.Contains(preflight.Header().Get("Access-Control-Allow-Headers"), "Idempotency-Key") {
		t.Errorf("preflight = %d %v", preflight.Code, preflight.Header())
	}
}
