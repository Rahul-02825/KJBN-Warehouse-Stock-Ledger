package test

import (
	"fmt"
	"testing"
)

func stockTake(lines string) string {
	return `{"location_id":1,"counts":[` + lines + `],"created_by":"t"}`
}

func adjustmentsBy(t *testing.T, body map[string]any) map[float64]string {
	t.Helper()
	out := map[float64]string{}
	for _, a := range body["adjustments"].([]any) {
		m := a.(map[string]any)
		sign := "+"
		if m["from_location_id"] != nil {
			sign = "-"
		}
		out[m["product_id"].(float64)] = sign + m["quantity"].(string)
	}
	return out
}

func TestStockTake_PostsOnlyTheDifferences(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 10))
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveKg("1", "5.5"))

	body := mustSend(t, h, 201, "POST", "/api/stock-takes", stockTake(`{"product_id":1,"counted":"8"},{"product_id":3,"counted":"5.5"},{"product_id":2,"counted":"3"}`))
	got := adjustmentsBy(t, body)
	if got[1] != "-2" || got[2] != "+3" || len(got) != 2 || body["unchanged"] != float64(1) {
		t.Fatalf("adjustments = %v, unchanged = %v; want RICE -2, WHEAT +3, DAL unchanged", got, body["unchanged"])
	}
	if stockAt(t, h, 1, 1) != "8" || stockAt(t, h, 2, 1) != "3" || stockAt(t, h, 3, 1) != "5.5" {
		t.Error("balances should equal the counted quantities")
	}
	detail := mustSend(t, h, 200, "GET", "/api/movements/"+formatID(body["adjustments"].([]any)[0]), "")
	if detail["reason"] != "count_correction" || detail["reference"] != "Stock take" {
		t.Errorf("adjustment = %v, want reason count_correction and reference Stock take", detail)
	}
}

func TestStockTake_SubmittingTheSameCountTwicePostsNothingNew(t *testing.T) {
	h, _ := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 10))
	count := stockTake(`{"product_id":1,"counted":"7"}`)
	mustSend(t, h, 201, "POST", "/api/stock-takes", count)

	again := mustSend(t, h, 201, "POST", "/api/stock-takes", count)
	if n := len(again["adjustments"].([]any)); n != 0 || again["unchanged"] != float64(1) {
		t.Errorf("second identical count posted %d adjustments, want 0", n)
	}
}

func TestStockTake_IsAllOrNothing(t *testing.T) {
	h, pool := setupLayout(t)
	mustSend(t, h, 201, "POST", "/api/movements/in", receiveInto(1, "1", 10))
	before := countMovements(t, pool)

	cases := map[string]string{
		stockTake(`{"product_id":1,"counted":"8"},{"product_id":2,"counted":"1.5"}`): "VALIDATION_ERROR",
		stockTake(`{"product_id":1,"counted":"8"},{"product_id":99,"counted":"1"}`):  "NOT_FOUND",
		stockTake(`{"product_id":1,"counted":"8"},{"product_id":2,"counted":"500"}`): "NO_SPACE",
	}
	for body, code := range cases {
		if status, resp := send(t, h, "POST", "/api/stock-takes", body); errorCode(resp) != code {
			t.Errorf("%s: %d %v, want %s", body, status, resp, code)
		}
	}
	if countMovements(t, pool) != before || stockAt(t, h, 1, 1) != "10" {
		t.Error("a failed count must not post any adjustment")
	}
}

func TestStockTake_RejectsBadInput(t *testing.T) {
	h := routerWithoutDB()
	for _, body := range []string{
		`{"location_id":1,"counts":[],"created_by":"t"}`,
		`{"location_id":0,"counts":[{"product_id":1,"counted":"1"}],"created_by":"t"}`,
		`{"location_id":1,"counts":[{"product_id":1,"counted":"-1"}],"created_by":"t"}`,
		`{"location_id":1,"counts":[{"product_id":1,"counted":"1"},{"product_id":1,"counted":"2"}],"created_by":"t"}`,
		`{"location_id":1,"counts":[{"product_id":1,"counted":"1"}]}`,
	} {
		if status, resp := send(t, h, "POST", "/api/stock-takes", body); status != 400 || errorCode(resp) != "VALIDATION_ERROR" {
			t.Errorf("%s: %d %v, want 400", body, status, resp)
		}
	}
}

func formatID(movement any) string {
	return fmt.Sprint(movement.(map[string]any)["id"])
}
