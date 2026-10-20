package test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func TestLogBuffer_KeepsOnlyTheLast500Lines(t *testing.T) {
	buf := httpapi.NewLogBuffer()
	for i := 0; i < 510; i++ {
		fmt.Fprintf(buf, "{\"n\":%d}\n", i)
	}
	lines := buf.Lines()
	if len(lines) != 500 || string(lines[0]) != `{"n":10}` || string(lines[499]) != `{"n":509}` {
		t.Fatalf("kept %d lines, first %s, last %s", len(lines), lines[0], lines[len(lines)-1])
	}
}

func TestRecentLogs_FiltersByLevelAndRequestID(t *testing.T) {
	buf := httpapi.NewLogBuffer()
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	h := httpapi.NewRouter(nil, httpapi.Config{Logs: buf})

	rejected := sendRaw(h, "GET", "/api/movements?type=BAD", "", map[string]string{"X-Request-ID": "bad-id-1"})
	if rejected.Code != 400 {
		t.Fatalf("setup request: %d", rejected.Code)
	}
	sendRaw(h, "GET", "/api/movements?type=WRONG", "", map[string]string{"X-Request-ID": "bad-id-2"})

	_, all := send(t, h, "GET", "/api/logs", "")
	lines := all["data"].([]any)
	if len(lines) != 4 || lines[0].(map[string]any)["request_id"] != "bad-id-2" {
		t.Fatalf("want 4 lines newest first (2 requests x warning + request line), got %v", lines)
	}

	_, warnings := send(t, h, "GET", "/api/logs?level=WARN", "")
	for _, l := range warnings["data"].([]any) {
		if l.(map[string]any)["level"] != "WARN" {
			t.Errorf("level=WARN returned %v", l)
		}
	}
	if n := len(warnings["data"].([]any)); n != 2 {
		t.Errorf("level=WARN returned %d lines, want 2", n)
	}

	_, one := send(t, h, "GET", "/api/logs?request_id=bad-id-1", "")
	if n := len(one["data"].([]any)); n != 2 {
		t.Errorf("request_id filter returned %d lines, want 2", n)
	}

	_, limited := send(t, h, "GET", "/api/logs?limit=1", "")
	if n := len(limited["data"].([]any)); n != 1 {
		t.Errorf("limit=1 returned %d lines", n)
	}

	if status, body := send(t, h, "GET", "/api/logs?level=LOUD", ""); status != 400 || errorCode(body) != "VALIDATION_ERROR" {
		t.Errorf("bad level: %d %v", status, body)
	}

	for _, l := range lines {
		if l.(map[string]any)["path"] == "/api/logs" {
			t.Errorf("viewing the logs should not add log lines: %v", l)
		}
	}
}
