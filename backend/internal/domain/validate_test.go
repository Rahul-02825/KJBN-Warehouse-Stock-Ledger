package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidUnit(t *testing.T) {
	for _, unit := range []string{"pcs", "kg", "litre"} {
		if !ValidUnit(unit) {
			t.Errorf("%q should be valid", unit)
		}
	}
	for _, unit := range []string{"", "box", "PCS", "liter"} {
		if ValidUnit(unit) {
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
		msg := CheckReorderLevel(decimal.RequireFromString(tt.value), tt.unit)
		gotErr := msg != ""
		if gotErr != tt.wantErr {
			t.Errorf("CheckReorderLevel(%s, %s): got error=%v (%q), want error=%v",
				tt.value, tt.unit, gotErr, msg, tt.wantErr)
		}
	}
}

func TestValidateQuantity(t *testing.T) {
	ok := []string{"1", "0.0001", "1000000000"}
	bad := []string{"0", "-1", "0.00001", "1000000000.0001"}
	for _, s := range ok {
		if msg := ValidateQuantity(decimal.RequireFromString(s)); msg != "" {
			t.Errorf("%s should be valid, got %q", s, msg)
		}
	}
	for _, s := range bad {
		if ValidateQuantity(decimal.RequireFromString(s)) == "" {
			t.Errorf("%s should be invalid", s)
		}
	}
}

func TestValidReason(t *testing.T) {
	for _, r := range []string{"damaged", "expired", "count_correction", "theft", "other"} {
		if !ValidReason(r) {
			t.Errorf("%q should be valid", r)
		}
	}
	for _, r := range []string{"", "lost", "Damaged", "count correction"} {
		if ValidReason(r) {
			t.Errorf("%q should be invalid", r)
		}
	}
}
