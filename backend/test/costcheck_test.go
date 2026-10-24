package test

import (
	"context"
	"fmt"
	"testing"
)

func TestCostCheck_MatchesAfterEveryMovementType(t *testing.T) {
	h, pool := setupWarehouse(t)
	receive(t, h, 1, 1, "10", "100")
	r2 := receive(t, h, 1, 1, "10", "130")
	mustSend(t, h, 201, "POST", fmt.Sprintf("/api/movements/%v/reverse", r2["id"]), `{"created_by":"t"}`)
	receive(t, h, 1, 1, "10", "130")
	mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"3","created_by":"t"}`)
	mustSend(t, h, 201, "POST", "/api/movements/transfer", `{"product_id":1,"from_location_id":1,"to_location_id":2,"quantity":"2","created_by":"t"}`)
	mustSend(t, h, 201, "POST", "/api/movements/adjustment", `{"product_id":1,"location_id":2,"quantity":"-1","reason":"damaged","created_by":"t"}`)
	receive(t, h, 1, 1, "7", "97.3333")
	out := mustSend(t, h, 201, "POST", "/api/movements/out", `{"product_id":1,"location_id":1,"quantity":"2","created_by":"t"}`)
	mustSend(t, h, 201, "POST", fmt.Sprintf("/api/movements/%v/reverse", out["id"]), `{"created_by":"t"}`)
	receive(t, h, 2, 2, "1.5", "40")

	body := mustSend(t, h, 200, "GET", "/api/reports/cost-check", "")
	if body["mismatch_count"] != float64(0) {
		t.Fatalf("mismatch_count = %v, rows %v", body["mismatch_count"], body["data"])
	}

	// A wrong stored average is caught even though every quantity still reconciles.
	if _, err := pool.Exec(context.Background(), "UPDATE products SET average_cost = 1 WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	body = mustSend(t, h, 200, "GET", "/api/reports/cost-check", "")
	row := body["data"].([]any)[1].(map[string]any) // ordered by SKU: DAL, RICE-25
	if body["mismatch_count"] != float64(1) || row["sku"] != "RICE-25" || row["matches"] != false {
		t.Fatalf("after corruption: %v", body)
	}
	recomputed := row["recomputed_average"].(string)
	if _, err := pool.Exec(context.Background(), "UPDATE products SET average_cost = $1 WHERE id = 1", recomputed); err != nil {
		t.Fatal(err)
	}
}
