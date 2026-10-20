package domain

import (
	"slices"
	"strings"

	"github.com/shopspring/decimal"
)

func ValidateQuantity(d decimal.Decimal) string {
	switch {
	case !d.IsPositive():
		return "must be greater than zero"
	case !d.Equal(d.Round(Decimals)):
		return "must have at most 4 decimal places"
	case d.GreaterThan(MaxAmount):
		return "must not exceed " + MaxAmount.String()
	}
	return ""
}

func ValidateCount(d decimal.Decimal) string {
	switch {
	case d.IsNegative():
		return "counted must not be negative"
	case !d.Equal(d.Round(Decimals)):
		return "counted must have at most 4 decimal places"
	case d.GreaterThan(MaxAmount):
		return "counted must not exceed " + MaxAmount.String()
	}
	return ""
}

func ValidUnit(u string) bool { return slices.Contains(Units, u) }

func UnitsMessage() string { return "must be one of " + strings.Join(Units, ", ") }

func IsWholeUnit(unit string) bool { return slices.Contains(WholeUnits, unit) }

func CheckWholeQuantity(d decimal.Decimal, unit string) string {
	if IsWholeUnit(unit) && !d.Equal(d.Truncate(0)) {
		return "must be a whole number for " + unit
	}
	return ""
}

func CheckReorderLevel(d decimal.Decimal, unit string) string {
	if d.IsNegative() {
		return "must not be negative"
	}
	if !d.Equal(d.Round(Decimals)) {
		return "must have at most 4 decimal places"
	}
	return CheckWholeQuantity(d, unit)
}

func ValidReason(r string) bool { return slices.Contains(AdjustmentReasons, r) }

func ReasonsMessage() string { return "must be one of " + strings.Join(AdjustmentReasons, ", ") }

func NeedsNote(reason string) bool { return slices.Contains(ReasonsNeedingNote, reason) }

func ValidMovementType(t string) bool { return slices.Contains(MovementTypes, t) }

func MovementTypesMessage() string { return "must be one of " + strings.Join(MovementTypes, ", ") }

func CheckReversible(movementType string) error {
	if movementType == "REVERSAL" {
		return NotReversibleError{"a reversal cannot itself be reversed"}
	}
	return nil
}
