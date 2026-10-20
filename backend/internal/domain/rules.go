package domain

import "github.com/shopspring/decimal"

const Decimals = 4

var MaxAmount = decimal.New(1, 9)

var Units = []string{"pcs", "kg", "litre"}

var WholeUnits = []string{"pcs"}

var MovementTypes = []string{"IN", "OUT", "TRANSFER", "ADJUSTMENT", "REVERSAL"}

var AdjustmentReasons = []string{"damaged", "expired", "count_correction", "theft", "other"}

var ReasonsNeedingNote = []string{"other"}

const StockTakeReason = "count_correction"

const MaxCountLines = 500

const DefaultStockTakeReference = "Stock take"
