package domain

import "github.com/shopspring/decimal"

const UnitScale = 10000

func ToUnits(d decimal.Decimal) int64 { return d.Shift(Decimals).IntPart() }

func FromUnits(n int64) decimal.Decimal { return decimal.New(n, -Decimals) }

func PlacementStep(unit string) int64 {
	if IsWholeUnit(unit) {
		return UnitScale
	}
	return 1
}
