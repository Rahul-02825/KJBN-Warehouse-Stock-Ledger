package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type lowStockJSON struct {
	ProductID     int64           `json:"product_id"`
	SKU           string          `json:"sku"`
	Name          string          `json:"name"`
	Unit          string          `json:"unit"`
	ReorderLevel  decimal.Decimal `json:"reorder_level"`
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	Shortfall     decimal.Decimal `json:"shortfall"`
}

// lowStockReport lists products whose total stock across all bins is at or below the reorder level.
// ?format=csv downloads it as a spreadsheet.
func (s *Server) lowStockReport(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListLowStockProducts(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]lowStockJSON, len(rows))
	for i, p := range rows {
		data[i] = lowStockJSON{
			ProductID:     p.ProductID,
			SKU:           p.Sku,
			Name:          p.Name,
			Unit:          p.Unit,
			ReorderLevel:  p.ReorderLevel,
			TotalQuantity: p.TotalQuantity,
			Shortfall:     p.Shortfall,
		}
	}
	if r.URL.Query().Get("format") == "csv" {
		rows := make([][]string, len(data))
		for i, p := range data {
			rows[i] = []string{p.SKU, csvText(p.Name), p.Unit, p.ReorderLevel.String(), p.TotalQuantity.String(), p.Shortfall.String()}
		}
		writeCSV(w, "low-stock.csv", []string{"sku", "name", "unit", "reorder_level", "total_quantity", "shortfall"}, rows)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

type valuationJSON struct {
	ProductID     int64           `json:"product_id"`
	SKU           string          `json:"sku"`
	Name          string          `json:"name"`
	Unit          string          `json:"unit"`
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	AverageCost   decimal.Decimal `json:"average_cost"`
	TotalValue    decimal.Decimal `json:"total_value"`
}

// valuationReport is quantity x average cost per product, plus the grand total.
// ?format=csv downloads it as a spreadsheet.
func (s *Server) valuationReport(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.GetInventoryValuation(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	total := decimal.Zero
	data := make([]valuationJSON, len(rows))
	for i, p := range rows {
		data[i] = valuationJSON{
			ProductID:     p.ProductID,
			SKU:           p.Sku,
			Name:          p.Name,
			Unit:          p.Unit,
			TotalQuantity: p.TotalQuantity,
			AverageCost:   p.AverageCost,
			TotalValue:    p.TotalValue,
		}
		total = total.Add(data[i].TotalValue)
	}
	if r.URL.Query().Get("format") == "csv" {
		rows := make([][]string, len(data))
		for i, p := range data {
			rows[i] = []string{p.SKU, csvText(p.Name), p.Unit, p.TotalQuantity.String(), p.AverageCost.String(), p.TotalValue.String()}
		}
		rows = append(rows, []string{"TOTAL", "", "", "", "", total.String()})
		writeCSV(w, "valuation.csv", []string{"sku", "name", "unit", "total_quantity", "average_cost", "total_value"}, rows)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "total_value": total})
}

type reconciliationJSON struct {
	ProductID      int64           `json:"product_id"`
	SKU            string          `json:"sku"`
	LocationID     int64           `json:"location_id"`
	LocationCode   string          `json:"location_code"`
	CachedQuantity decimal.Decimal `json:"cached_quantity"`
	LedgerQuantity decimal.Decimal `json:"ledger_quantity"`
	Difference     decimal.Decimal `json:"difference"`
}

// reconciliationReport recomputes every balance from the ledger and compares it with
// the cached balance. It only reports; it never changes data.
// ?only_drift=true returns just the rows that differ.
func (s *Server) reconciliationReport(w http.ResponseWriter, r *http.Request) {
	onlyDrift := r.URL.Query().Get("only_drift") == "true"
	rows, err := s.q.ListReconciliation(r.Context(), onlyDrift)
	if err != nil {
		writeError(w, r, err)
		return
	}
	driftCount := 0
	data := make([]reconciliationJSON, len(rows))
	for i, c := range rows {
		data[i] = reconciliationJSON{
			ProductID:      c.ProductID,
			SKU:            c.Sku,
			LocationID:     c.LocationID,
			LocationCode:   c.LocationCode,
			CachedQuantity: c.CachedQuantity,
			LedgerQuantity: c.LedgerQuantity,
			Difference:     c.Difference,
		}
		if !data[i].Difference.IsZero() {
			driftCount++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "drift_count": driftCount})
}

type ledgerEntryJSON struct {
	MovementID      int64            `json:"movement_id"`
	CreatedAt       time.Time        `json:"created_at"`
	Type            string           `json:"type"`
	Reference       string           `json:"reference"`
	Reason          *string          `json:"reason"`
	CreatedBy       string           `json:"created_by"`
	LocationID      int64            `json:"location_id"`
	LocationCode    string           `json:"location_code"`
	QuantityDelta   decimal.Decimal  `json:"quantity_delta"`
	UnitCost        *decimal.Decimal `json:"unit_cost"`
	RunningTotal    decimal.Decimal  `json:"running_total"`
	RunningLocation decimal.Decimal  `json:"running_location_balance"`
}

// productLedger lists every stock change for a product, oldest first, with running
// balances, so any current quantity can be explained line by line.
func (s *Server) productLedger(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	product, err := s.findProduct(r, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	rows, err := s.q.GetProductLedger(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	data := make([]ledgerEntryJSON, len(rows))
	for i, e := range rows {
		data[i] = ledgerEntryJSON{
			MovementID:      e.MovementID,
			CreatedAt:       e.CreatedAt.UTC(),
			Type:            e.MovementType,
			Reference:       e.Reference,
			Reason:          e.Reason,
			CreatedBy:       e.CreatedBy,
			LocationID:      e.LocationID,
			LocationCode:    e.LocationCode,
			QuantityDelta:   e.QuantityDelta,
			UnitCost:        e.UnitCost,
			RunningTotal:    e.RunningTotal,
			RunningLocation: e.RunningLocationBalance,
		}
	}
	if r.URL.Query().Get("format") == "csv" {
		writeLedgerCSV(w, product, data)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": product.ID,
		"sku":        product.Sku,
		"unit":       product.Unit,
		"data":       data,
	})
}

func writeLedgerCSV(w http.ResponseWriter, product sqlc.Product, data []ledgerEntryJSON) {
	rows := make([][]string, len(data))
	for i, e := range data {
		rows[i] = []string{
			e.CreatedAt.Format(time.RFC3339), strconv.FormatInt(e.MovementID, 10), e.Type, e.LocationCode,
			e.QuantityDelta.String(), csvDecimal(e.UnitCost), e.RunningLocation.String(), e.RunningTotal.String(),
			csvText(e.Reference), csvPtr(e.Reason), csvText(e.CreatedBy),
		}
	}
	writeCSV(w, "ledger-"+product.Sku+".csv", []string{
		"posted_at", "movement_id", "type", "bin", "change", "unit_cost", "bin_balance", "total_balance", "reference", "reason", "created_by",
	}, rows)
}

func (s *Server) costCheckReport(w http.ResponseWriter, r *http.Request) {
	rows, mismatches, err := s.ledger.CostCheck(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "mismatch_count": mismatches})
}

type activityJSON struct {
	Day             string          `json:"day"`
	InCount         int64           `json:"in_count"`
	OutCount        int64           `json:"out_count"`
	TransferCount   int64           `json:"transfer_count"`
	AdjustmentCount int64           `json:"adjustment_count"`
	ReversalCount   int64           `json:"reversal_count"`
	InValue         decimal.Decimal `json:"in_value"`
	OutValue        decimal.Decimal `json:"out_value"`
}

// dailyActivityReport counts postings per day for the dashboard chart. ?days=30 (1 to 365).
func (s *Server) dailyActivityReport(w http.ResponseWriter, r *http.Request) {
	fields := map[string]string{}
	days := queryInt(fields, r.URL.Query(), "days", 30, 1, 365)
	if len(fields) > 0 {
		writeError(w, r, domain.ValidationError(fields))
		return
	}
	rows, err := s.q.GetDailyActivity(r.Context(), int32(days))
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]activityJSON, len(rows))
	for i, d := range rows {
		data[i] = activityJSON{
			Day:             d.Day.Format("2006-01-02"),
			InCount:         d.InCount,
			OutCount:        d.OutCount,
			TransferCount:   d.TransferCount,
			AdjustmentCount: d.AdjustmentCount,
			ReversalCount:   d.ReversalCount,
			InValue:         d.InValue,
			OutValue:        d.OutValue,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

type topMoverJSON struct {
	ProductID       int64           `json:"product_id"`
	SKU             string          `json:"sku"`
	Name            string          `json:"name"`
	Unit            string          `json:"unit"`
	Shipments       int64           `json:"shipments"`
	ShippedQuantity decimal.Decimal `json:"shipped_quantity"`
	ShippedValue    decimal.Decimal `json:"shipped_value"`
}

// topMoversReport ranks products by value shipped in the last ?days=30, top ?limit=5.
func (s *Server) topMoversReport(w http.ResponseWriter, r *http.Request) {
	fields := map[string]string{}
	days := queryInt(fields, r.URL.Query(), "days", 30, 1, 365)
	limit := queryInt(fields, r.URL.Query(), "limit", 5, 1, 50)
	if len(fields) > 0 {
		writeError(w, r, domain.ValidationError(fields))
		return
	}
	now := time.Now().In(ist)
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ist).AddDate(0, 0, -(days - 1))

	rows, err := s.q.ListTopMovers(r.Context(), sqlc.ListTopMoversParams{
		Since:   since,
		MaxRows: int32(limit),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]topMoverJSON, len(rows))
	for i, p := range rows {
		data[i] = topMoverJSON{
			ProductID:       p.ProductID,
			SKU:             p.Sku,
			Name:            p.Name,
			Unit:            p.Unit,
			Shipments:       p.Shipments,
			ShippedQuantity: p.ShippedQuantity,
			ShippedValue:    p.ShippedValue,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "since": since})
}

type forecastJSON struct {
	ProductID      int64           `json:"product_id"`
	SKU            string          `json:"sku"`
	Name           string          `json:"name"`
	Unit           string          `json:"unit"`
	LastYearDemand decimal.Decimal `json:"last_year_demand"`
	CurrentStock   decimal.Decimal `json:"current_stock"`
	Gap            decimal.Decimal `json:"gap"`
	AverageCost    decimal.Decimal `json:"average_cost"`
	GapValue       decimal.Decimal `json:"gap_value"`
}

// seasonForecastReport compares current stock with what was shipped in the same
// ?days=30 window last year, so stock can be bought before the season starts.
func (s *Server) seasonForecastReport(w http.ResponseWriter, r *http.Request) {
	fields := map[string]string{}
	days := queryInt(fields, r.URL.Query(), "days", 30, 1, 180)
	if len(fields) > 0 {
		writeError(w, r, domain.ValidationError(fields))
		return
	}
	start := time.Now().AddDate(-1, 0, 0)
	end := start.AddDate(0, 0, days)

	rows, err := s.q.ListSeasonForecast(r.Context(), sqlc.ListSeasonForecastParams{
		WindowStart: start,
		WindowEnd:   end,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]forecastJSON, len(rows))
	for i, p := range rows {
		data[i] = forecastJSON{
			ProductID:      p.ProductID,
			SKU:            p.Sku,
			Name:           p.Name,
			Unit:           p.Unit,
			LastYearDemand: p.LastYearDemand,
			CurrentStock:   p.CurrentStock,
			Gap:            p.Gap,
			AverageCost:    p.AverageCost,
			GapValue:       p.GapValue,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"window_start": start.UTC(),
		"window_end":   end.UTC(),
		"data":         data,
	})
}
