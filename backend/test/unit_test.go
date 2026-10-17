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
