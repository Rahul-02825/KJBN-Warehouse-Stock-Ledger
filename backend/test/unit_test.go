package test

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/api"
)

// Unit tests: plain functions, no HTTP and no database.

func TestValidUnit(t *testing.T) {
	for _, unit := range []string{"pcs", "kg", "litre"} {
		if !api.ValidUnit(unit) {
			t.Errorf("%q should be valid", unit)
		}
	}
	for _, unit := range []string{"", "box", "PCS", "liter"} {
		if api.ValidUnit(unit) {
			t.Errorf("%q should be invalid", unit)
		}
	}
}

func TestCheckReorderLevel(t *testing.T) {
	tests := []struct {
		value   string
		unit    string
		wantErr bool
	}{
		{"0", "pcs", false},
		{"10", "pcs", false},
		{"1.5", "pcs", true}, // pieces must be whole
		{"1.5", "kg", false},
		{"0.0001", "litre", false},
		{"0.00001", "kg", true}, // more than 4 decimals
		{"-1", "kg", true},
	}

	for _, tt := range tests {
		msg := api.CheckReorderLevel(decimal.RequireFromString(tt.value), tt.unit)
		gotErr := msg != ""
		if gotErr != tt.wantErr {
			t.Errorf("CheckReorderLevel(%s, %s): got error=%v (%q), want error=%v",
				tt.value, tt.unit, gotErr, msg, tt.wantErr)
		}
	}
}

func TestWeightedAverage(t *testing.T) {
	d := decimal.RequireFromString
	tests := []struct {
		name                      string
		oldQty, oldAvg, qty, cost string
		want                      string
	}{
		{"first receipt takes its own cost", "0", "0", "50", "1000", "1000"},
		{"two equal lots", "50", "1000", "50", "1200", "1100"},
		{"unequal lots", "6", "100", "10", "200", "162.5"},
		{"rounds to 4 decimals", "2", "10", "1", "11", "10.3333"},
	}
	for _, tt := range tests {
		got := api.WeightedAverage(d(tt.oldQty), d(tt.oldAvg), d(tt.qty), d(tt.cost))
		if !got.Equal(d(tt.want)) {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
		}
	}
}

// The average is stored with 4 decimals, so each receipt can add up to 0.00005 of
// rounding error. Over 1000 receipts the measured drift is about 0.00013 rupee;
// the test allows 0.001 rupee (a tenth of a paisa).
func TestWeightedAverage_RoundingAfterManyReceipts(t *testing.T) {
	avg, qty := decimal.Zero, decimal.Zero
	totalValue := decimal.Zero
	for i := 1; i <= 1000; i++ {
		lot := decimal.NewFromInt(int64(i%7 + 1))
		cost := decimal.NewFromInt(int64(i%13 + 1)).Div(decimal.NewFromInt(3)).Round(4)
		avg = api.WeightedAverage(qty, avg, lot, cost)
		qty = qty.Add(lot)
		totalValue = totalValue.Add(lot.Mul(cost))
	}
	exact := totalValue.DivRound(qty, 10)
	if diff := avg.Sub(exact).Abs(); diff.GreaterThan(decimal.RequireFromString("0.001")) {
		t.Fatalf("average drifted: rounded %s, exact %s, diff %s", avg, exact, diff)
	}
}

func TestValidateQuantity(t *testing.T) {
	ok := []string{"1", "0.0001", "1000000000"}
	bad := []string{"0", "-1", "0.00001", "1000000000.0001"}
	for _, s := range ok {
		if msg := api.ValidateQuantity(decimal.RequireFromString(s)); msg != "" {
			t.Errorf("%s should be valid, got %q", s, msg)
		}
	}
	for _, s := range bad {
		if api.ValidateQuantity(decimal.RequireFromString(s)) == "" {
			t.Errorf("%s should be invalid", s)
		}
	}
}

func TestValidReason(t *testing.T) {
	for _, r := range []string{"damaged", "expired", "count_correction", "theft", "other"} {
		if !api.ValidReason(r) {
			t.Errorf("%q should be valid", r)
		}
	}
	for _, r := range []string{"", "lost", "Damaged", "count correction"} {
		if api.ValidReason(r) {
			t.Errorf("%q should be invalid", r)
		}
	}
}

func TestBackOutAverage(t *testing.T) {
	d := decimal.RequireFromString
	// 10 @ 100 then 10 @ 130 gives 115. Backing out the second receipt returns to 100.
	got, ok := api.BackOutAverage(d("10"), d("115"), d("10"), d("130"))
	if !ok || !got.Equal(d("100")) {
		t.Errorf("got %s %v, want 100", got, ok)
	}
	// Nothing left: keep the last average.
	if got, ok := api.BackOutAverage(d("0"), d("115"), d("10"), d("130")); !ok || !got.Equal(d("115")) {
		t.Errorf("empty: got %s %v, want 115", got, ok)
	}
	// The cheap stock is gone, so removing the expensive receipt would make value negative.
	if _, ok := api.BackOutAverage(d("1"), d("115"), d("10"), d("130")); ok {
		t.Error("expected not reversible")
	}
}
