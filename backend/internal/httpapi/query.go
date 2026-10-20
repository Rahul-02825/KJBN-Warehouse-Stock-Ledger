package httpapi

import (
	"net/url"
	"strconv"
	"time"
)

var ist = time.FixedZone("IST", 5*3600+1800)

func pageParams(query url.Values) (int, int) {
	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(query.Get("page_size"))
	if err != nil || size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func queryID(fields map[string]string, query url.Values, name string) *int64 {
	raw := query.Get(name)
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		fields[name] = "must be a positive integer"
		return nil
	}
	return &id
}

func queryTime(fields map[string]string, query url.Values, name string, endOfDay bool) *time.Time {
	raw := query.Get(name)
	if raw == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t
	}
	t, err := time.ParseInLocation("2006-01-02", raw, ist)
	if err != nil {
		fields[name] = "must be a date (2006-01-02) or RFC 3339 timestamp"
		return nil
	}
	if endOfDay {
		t = t.AddDate(0, 0, 1)
	}
	return &t
}

func queryInt(fields map[string]string, query url.Values, name string, defaultValue, min, max int) int {
	raw := query.Get(name)
	if raw == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min || n > max {
		fields[name] = "must be a whole number from " + strconv.Itoa(min) + " to " + strconv.Itoa(max)
		return defaultValue
	}
	return n
}
