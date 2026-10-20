package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
)

var levelRank = map[string]int{"DEBUG": 0, "INFO": 1, "WARN": 2, "ERROR": 3}

func (s *Server) recentLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fields := map[string]string{}
	minLevel := strings.ToUpper(query.Get("level"))
	if _, known := levelRank[minLevel]; minLevel != "" && !known {
		fields["level"] = "must be one of INFO, WARN, ERROR"
	}
	limit := queryInt(fields, query, "limit", 100, 1, logBufferSize)
	if len(fields) > 0 {
		writeError(w, r, domain.ValidationError(fields))
		return
	}
	requestID := query.Get("request_id")

	lines := []map[string]any{}
	if s.logs != nil {
		all := s.logs.Lines()
		for i := len(all) - 1; i >= 0 && len(lines) < limit; i-- {
			var line map[string]any
			if err := json.Unmarshal(all[i], &line); err != nil {
				continue
			}
			level, _ := line["level"].(string)
			if minLevel != "" && levelRank[level] < levelRank[minLevel] {
				continue
			}
			if requestID != "" && line["request_id"] != requestID {
				continue
			}
			lines = append(lines, line)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": lines, "capacity": logBufferSize})
}
