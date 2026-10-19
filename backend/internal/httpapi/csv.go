package httpapi

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

const csvMaxRows = 100000

func writeCSV(w http.ResponseWriter, filename string, header []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	cw := csv.NewWriter(w)
	cw.Write(header)
	cw.WriteAll(rows)
}

func csvText(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func csvPtr(s *string) string {
	if s == nil {
		return ""
	}
	return csvText(*s)
}

func csvDecimal(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}

func csvID(id *int64) string {
	if id == nil {
		return ""
	}
	return strconv.FormatInt(*id, 10)
}
