package test

import (
	"fmt"
	"testing"
)

func TestDailyActivityAndTopMovers(t *testing.T) {
	h, _ := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")
	receive(t, h, 2, 1, "10", "40")
	mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"2","created_by":"t"}`)
	mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":2,"location_id":1,"quantity":"1","created_by":"t"}`)
	big := mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":2,"location_id":1,"quantity":"9","created_by":"t"}`)
	mustSend(t, h, 201, "POST", fmt.Sprintf("/api/movements/%v/reverse", big["id"]), `{"created_by":"t"}`)

	body := mustSend(t, h, 200, "GET", "/api/reports/daily-activity?days=7", "")
	days := body["data"].([]any)
	today := days[len(days)-1].(map[string]any)
	if len(days) != 7 || today["in_count"] != float64(2) || today["out_count"] != float64(3) || today["reversal_count"] != float64(1) {
		t.Fatalf("daily activity = %v", days)
	}
	if today["in_value"] != "1400" || today["out_value"] != "600" {
		t.Errorf("values = %v / %v, want 1400 / 600", today["in_value"], today["out_value"])
	}

	// RICE shipped 200; DAL shipped 40 (the reversed 360 does not count).
	body = mustSend(t, h, 200, "GET", "/api/reports/top-movers", "")
	movers := body["data"].([]any)
	if len(movers) != 2 || movers[0].(map[string]any)["sku"] != "RICE-25" || movers[1].(map[string]any)["shipped_value"] != "40" {
		t.Fatalf("top movers = %v", movers)
	}

	if status, _ := send(t, h, "GET", "/api/reports/daily-activity?days=0", ""); status != 400 {
		t.Errorf("days=0: %d, want 400", status)
	}
	if status, _ := send(t, h, "GET", "/api/reports/top-movers?limit=500", ""); status != 400 {
		t.Errorf("limit=500: %d, want 400", status)
	}
}
