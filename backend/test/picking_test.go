package test

import (
	"fmt"
	"sync"
	"testing"
)

func ship(product int, location string, qty string) string {
	bin := ""
	if location != "" {
		bin = fmt.Sprintf(`"location_id":%s,`, location)
	}
	return fmt.Sprintf(`{"product_id":%d,%s"quantity":"%s","created_by":"t"}`, product, bin, qty)
}

func TestShip_PicksFromNearestBinsHoldingTheProduct(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 30))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "3", 20))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "2", 5))

	m := mustSend(t, h, 201, "POST", "/api/movements/out", ship(1, "1", "40"))
	if m["from_location_id"] != float64(1) || m["total_value"] != "400" {
		t.Errorf("movement = %v, want from A-01 valued at 40 x 10", m)
	}
	got := entriesByBin(t, h, m["id"])
	if got[1] != "-30" || got[2] != "-5" || got[3] != "-5" || len(got) != 3 {
		t.Fatalf("entries = %v, want A-01 -30, A-02 -5 (6 m), B-01 -5 (10 m)", got)
	}
	if stockAt(t, h, 1, 1) != "0" || stockAt(t, h, 1, 2) != "0" || stockAt(t, h, 1, 3) != "15" {
		t.Error("balances do not match the pick")
	}
}

func TestShip_DefaultStartIsTheBinHoldingTheMost(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 5))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "3", 20))

	m := mustSend(t, h, 201, "POST", "/api/movements/out", ship(1, "", "8"))
	if got := entriesByBin(t, h, m["id"]); got[3] != "-8" || len(got) != 1 {
		t.Errorf("entries = %v, want all 8 from B-01, which holds the most", got)
	}
}

func TestShip_RefusesWhenTheWarehouseIsShort(t *testing.T) {
	h, pool := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 30))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "3", 25))
	before := countMovements(t, pool)

	status, body := send(t, h, "POST", "/api/movements/out", ship(1, "1", "56"))
	details, _ := body["error"].(map[string]any)["details"].(map[string]any)
	if status != 409 || errorCode(body) != "INSUFFICIENT_STOCK" || details["available"] != "55" {
		t.Fatalf("got %d %v, want 409 with 55 available", status, body)
	}
	if countMovements(t, pool) != before {
		t.Error("a refused shipment must not post anything")
	}
}

func TestShip_StartingFromABinWithoutTheProduct(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "3", 10))
	m := mustSend(t, h, 201, "POST", "/api/movements/out", ship(1, "1", "4"))
	if m["from_location_id"] != float64(3) {
		t.Errorf("from = %v, want B-01, the only bin that supplied stock", m["from_location_id"])
	}
}

func TestShip_ReversingAMultiBinShipmentRestoresEachBin(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 30))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "3", 20))
	m := mustSend(t, h, 201, "POST", "/api/movements/out", ship(1, "1", "45"))

	mustSend(t, h, 201, "POST", fmt.Sprintf("/api/movements/%v/reverse", m["id"]), `{"created_by":"t"}`)
	if stockAt(t, h, 1, 1) != "30" || stockAt(t, h, 1, 3) != "20" {
		t.Errorf("after reversal A-01 = %s, B-01 = %s; want 30 and 20", stockAt(t, h, 1, 1), stockAt(t, h, 1, 3))
	}
}

func TestShip_ConcurrentPicksAcrossBinsNeverGoNegative(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 3))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "3", 2))

	var mu sync.Mutex
	var wg sync.WaitGroup
	codes := map[int]int{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _ := send(t, h, "POST", "/api/movements/out", ship(1, "", "1"))
			mu.Lock()
			codes[status]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if codes[201] != 5 || codes[409] != 15 {
		t.Fatalf("codes = %v, want 5 shipped and 15 refused", codes)
	}
	if stockAt(t, h, 1, 1) != "0" || stockAt(t, h, 1, 3) != "0" {
		t.Error("both bins should be empty, never negative")
	}
}

func TestPickPreview(t *testing.T) {
	h, pool := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveKg("1", "12.5"))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveKg("3", "4.25"))
	before := countMovements(t, pool)

	body := mustSend(t, h, 200, "POST", "/api/picks/preview", `{"product_id":3,"quantity":"15","location_id":1}`)
	allocations := body["allocations"].([]any)
	second := allocations[1].(map[string]any)
	if body["unit"] != "kg" || len(allocations) != 2 || second["location_code"] != "B-01" || second["quantity"] != "2.5" {
		t.Errorf("preview = %v, want A-01 12.5 then B-01 2.5", body)
	}
	if countMovements(t, pool) != before {
		t.Error("preview must not post")
	}
	if status, body := send(t, h, "POST", "/api/picks/preview", `{"product_id":3,"quantity":"17"}`); status != 409 || errorCode(body) != "INSUFFICIENT_STOCK" {
		t.Errorf("too much: %d %v", status, body)
	}
}
