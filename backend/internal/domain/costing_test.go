package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

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
		got := WeightedAverage(d(tt.oldQty), d(tt.oldAvg), d(tt.qty), d(tt.cost))
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
		avg = WeightedAverage(qty, avg, lot, cost)
		qty = qty.Add(lot)
		totalValue = totalValue.Add(lot.Mul(cost))
	}
	exact := totalValue.DivRound(qty, 10)
	if diff := avg.Sub(exact).Abs(); diff.GreaterThan(decimal.RequireFromString("0.001")) {
		t.Fatalf("average drifted: rounded %s, exact %s, diff %s", avg, exact, diff)
	}
}

func TestBackOutAverage(t *testing.T) {
	d := decimal.RequireFromString
	// 10 @ 100 then 10 @ 130 gives 115. Backing out the second receipt returns to 100.
	got, ok := BackOutAverage(d("10"), d("115"), d("10"), d("130"))
	if !ok || !got.Equal(d("100")) {
		t.Errorf("got %s %v, want 100", got, ok)
	}
	// Nothing left: keep the last average.
	if got, ok := BackOutAverage(d("0"), d("115"), d("10"), d("130")); !ok || !got.Equal(d("115")) {
		t.Errorf("empty: got %s %v, want 115", got, ok)
	}
	// The cheap stock is gone, so removing the expensive receipt would make value negative.
	if _, ok := BackOutAverage(d("1"), d("115"), d("10"), d("130")); ok {
		t.Error("expected not reversible")
	}
}
