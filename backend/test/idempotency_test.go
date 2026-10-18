package test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestIdempotency_RetryReturnsFirstMovement(t *testing.T) {
	h, _ := setupWarehouse(t)
	in := `{"product_id":1,"location_id":1,"quantity":"10","unit_cost":"100","created_by":"t"}`

	first := sendRaw(h, "POST", "/api/movements/in", in, withKey("abc-1"))
	retry := sendRaw(h, "POST", "/api/movements/in", in, withKey("abc-1"))
	if first.Code != 201 || retry.Code != 200 {
		t.Fatalf("first %d, retry %d; want 201 then 200", first.Code, retry.Code)
	}
	if retry.Header().Get("Idempotent-Replayed") != "true" {
		t.Error("retry should carry Idempotent-Replayed: true")
	}
	if first.Body.String() != retry.Body.String() {
		t.Errorf("retry body differs:\n%s\n%s", first.Body, retry.Body)
	}
	if got := stockAt(t, h, 1, 1); got != "10" {
		t.Errorf("stock = %s, want 10 (posted once)", got)
	}

	// Same key on a different kind of request is a client bug, not a replay.
	rec := sendRaw(h, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"1","created_by":"t"}`, withKey("abc-1"))
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), "IDEMPOTENCY_KEY_REUSED") {
		t.Errorf("reused key: %d %s", rec.Code, rec.Body)
	}
}

func TestIdempotency_ConcurrentRetriesPostOnce(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")

	var mu sync.Mutex
	var wg sync.WaitGroup
	codes := map[int]int{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := sendRaw(h, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"3","created_by":"t"}`, withKey("ship-1"))
			mu.Lock()
			codes[rec.Code]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if codes[201] != 1 || codes[200] != 9 {
		t.Fatalf("codes = %v, want one 201 and nine 200", codes)
	}
	if got := stockAt(t, h, 1, 1); got != "7" {
		t.Errorf("stock = %s, want 7 (shipped once)", got)
	}
}

func TestIdempotency_ReversalRetry(t *testing.T) {
	h, _ := setupWarehouse(t)
	m := receive(t, h, 1, 1, "10", "100")
	path := fmt.Sprintf("/api/movements/%v/reverse", m["id"])

	if rec := sendRaw(h, "POST", path, `{"created_by":"t"}`, withKey("rev-1")); rec.Code != 201 {
		t.Fatalf("reverse: %d %s", rec.Code, rec.Body)
	}
	if rec := sendRaw(h, "POST", path, `{"created_by":"t"}`, withKey("rev-1")); rec.Code != 200 {
		t.Errorf("retry with same key: %d, want 200 replay", rec.Code)
	}
	if status, body := send(t, h, "POST", path, `{"created_by":"t"}`); status != 409 || errorCode(body) != "ALREADY_REVERSED" {
		t.Errorf("new reversal without key: %d %q, want 409 ALREADY_REVERSED", status, errorCode(body))
	}
}

func TestIdempotency_KeyTooLong(t *testing.T) {
	rec := sendRaw(routerWithoutDB(), "POST", "/api/movements/in", `{}`, withKey(strings.Repeat("k", 129)))
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "Idempotency-Key") {
		t.Errorf("got %d %s, want 400 naming Idempotency-Key", rec.Code, rec.Body)
	}
}
