package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// backdate moves a movement and its ledger entries back in time, to fake last year's history.
func backdate(t *testing.T, pool *pgxpool.Pool, movementID any, ago string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"UPDATE movements SET created_at = NOW() - $2::INTERVAL WHERE id = $1", movementID, ago)
	if err == nil {
		_, err = pool.Exec(context.Background(),
			"UPDATE movement_entries SET created_at = NOW() - $2::INTERVAL WHERE movement_id = $1", movementID, ago)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestSeasonForecast_ComparesStockWithLastYearDemand(t *testing.T) {
	h, pool := setupWarehouse(t)
	receive(t, h, 1, 1, "100", "100")
	receive(t, h, 2, 1, "50", "40")
	out := func(product int, qty string) map[string]any {
		return mustSend(t, h, 201, "POST", "/api/movements/out",
			fmt.Sprintf(`{"product_id":%d,"location_id":1,"quantity":"%s","created_by":"t"}`, product, qty))
	}

	// The default window is the 30 days that started exactly one year ago.
	m := out(1, "30") // RICE, counts
	backdate(t, pool, m["id"], "1 year -5 days")
	m = out(1, "10") // RICE, reversed later, does not count
	backdate(t, pool, m["id"], "1 year -10 days")
	mustSend(t, h, 201, "POST", fmt.Sprintf("/api/movements/%v/reverse", m["id"]), `{"created_by":"t"}`)
	m = out(1, "20") // RICE, two years ago, does not count
	backdate(t, pool, m["id"], "2 years")
	m = out(2, "45") // DAL, counts
	backdate(t, pool, m["id"], "1 year -15 days")

	// Stock now: RICE 100-30-10+10-20 = 50, DAL 50-45 = 5.
	// RICE demand 30 is covered by 50; DAL demand 45 leaves a gap of 40 (x 40 = 1600).
	body := mustSend(t, h, 200, "GET", "/api/reports/season-forecast", "")
	rows := body["data"].([]any)
	if len(rows) != 1 {
		t.Fatalf("got %v, want only DAL", rows)
	}
	dal := rows[0].(map[string]any)
	if dal["sku"] != "DAL" || dal["last_year_demand"] != "45" || dal["current_stock"] != "5" ||
		dal["gap"] != "40" || dal["gap_value"] != "1600" {
		t.Errorf("DAL row = %v", dal)
	}

	// Ship 25 RICE today: 25 left is below last year's 30, so RICE is listed after DAL (500 < 1600).
	out(1, "25")
	body = mustSend(t, h, 200, "GET", "/api/reports/season-forecast", "")
	rows = body["data"].([]any)
	if len(rows) != 2 || rows[0].(map[string]any)["sku"] != "DAL" ||
		rows[1].(map[string]any)["gap"] != "5" || rows[1].(map[string]any)["gap_value"] != "500" {
		t.Errorf("want DAL then RICE (gap 5, value 500), got %v", rows)
	}

	// A 4-day window ends before the earliest counted shipment (day 5), so nothing is listed.
	body = mustSend(t, h, 200, "GET", "/api/reports/season-forecast?days=4", "")
	if rows := body["data"].([]any); len(rows) != 0 {
		t.Errorf("days=4: got %v, want nothing", rows)
	}
}

func TestSeasonForecast_BadDays(t *testing.T) {
	h := routerWithoutDB()
	for _, days := range []string{"0", "181", "x"} {
		status, body := send(t, h, "GET", "/api/reports/season-forecast?days="+days, "")
		if status != 400 || errorCode(body) != "VALIDATION_ERROR" {
			t.Errorf("days=%s: %d %q, want 400 VALIDATION_ERROR", days, status, errorCode(body))
		}
	}
}
