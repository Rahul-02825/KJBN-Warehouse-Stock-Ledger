package domain

import "github.com/shopspring/decimal"

func WeightedAverage(oldQty, oldAvg, qty, cost decimal.Decimal) decimal.Decimal {
	if !oldQty.IsPositive() {
		return cost.Round(Decimals)
	}
	totalValue := oldQty.Mul(oldAvg).Add(qty.Mul(cost))
	return totalValue.Div(oldQty.Add(qty)).Round(Decimals)
}

func BackOutAverage(remaining, avg, qty, cost decimal.Decimal) (decimal.Decimal, bool) {
	if !remaining.IsPositive() {
		return avg, true
	}
	value := remaining.Add(qty).Mul(avg).Sub(qty.Mul(cost))
	if value.IsNegative() {
		return avg, false
	}
	return value.Div(remaining).Round(Decimals), true
}
