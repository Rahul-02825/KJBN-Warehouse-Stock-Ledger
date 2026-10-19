// Command loadtest fires concurrent requests at a running API and checks that
// stock never goes negative and that a retried posting is applied only once.
//
//	go run ./cmd/loadtest -url http://localhost:8080 -stock 25 -orders 100
//
// It creates its own product and bin, so it is safe to run against a dev database,
// but it does write data. Do not point it at production.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

var baseURL string

func main() {
	flag.StringVar(&baseURL, "url", "http://localhost:8080", "API base URL")
	stock := flag.Int("stock", 25, "units received before the rush")
	orders := flag.Int("orders", 100, "concurrent OUT requests of 1 unit each")
	retries := flag.Int("retries", 20, "concurrent copies of one IN with the same Idempotency-Key")
	flag.Parse()

	run := time.Now().Format("150405")
	product := create("/api/products", map[string]any{"sku": "LOAD-" + run, "name": "Load test item", "unit": "pcs"})
	location := create("/api/locations", map[string]any{"code": "LOAD-" + run})
	post("/api/movements/in", "", map[string]any{
		"product_id": product, "location_id": location, "quantity": *stock, "unit_cost": "10", "created_by": "loadtest",
	})

	fmt.Printf("1) %d concurrent OUTs of 1 unit against %d in stock\n", *orders, *stock)
	counts := fire(*orders, func() int {
		status, _ := post("/api/movements/out", "", map[string]any{
			"product_id": product, "location_id": location, "quantity": 1, "created_by": "loadtest",
		})
		return status
	})
	fmt.Printf("   201 shipped: %d, 409 rejected: %d, other: %d\n", counts[201], counts[409], *orders-counts[201]-counts[409])
	left := quantity(product, location)
	fmt.Printf("   stock left: %s\n", left)
	ok := counts[201] == min(*stock, *orders) && left == fmt.Sprint(max(*stock-*orders, 0))

	fmt.Printf("2) %d concurrent copies of one IN with the same Idempotency-Key\n", *retries)
	key := "loadtest-" + run
	counts = fire(*retries, func() int {
		status, _ := post("/api/movements/in", key, map[string]any{
			"product_id": product, "location_id": location, "quantity": 5, "unit_cost": "10", "created_by": "loadtest",
		})
		return status
	})
	fmt.Printf("   201 posted: %d, 200 replayed: %d\n", counts[201], counts[200])
	after := quantity(product, location)
	fmt.Printf("   stock now: %s\n", after)
	ok = ok && counts[201] == 1 && counts[200] == *retries-1 && after == fmt.Sprint(max(*stock-*orders, 0)+5)

	_, recon := get("/api/reports/reconciliation?only_drift=true")
	fmt.Printf("3) reconciliation drift_count: %v\n", recon["drift_count"])
	ok = ok && recon["drift_count"] == float64(0)

	if !ok {
		fmt.Println("FAIL")
		os.Exit(1)
	}
	fmt.Println("PASS")
}

// fire runs fn n times at once and counts the status codes.
func fire(n int, fn func() int) map[int]int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	counts := map[int]int{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := fn()
			mu.Lock()
			counts[status]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	return counts
}

func create(path string, body map[string]any) float64 {
	status, out := post(path, "", body)
	if status != 201 {
		fail("POST %s: %d %v", path, status, out)
	}
	return out["id"].(float64)
}

func quantity(product, location float64) string {
	_, out := get(fmt.Sprintf("/api/inventory/%.0f/%.0f", product, location))
	return fmt.Sprint(out["quantity"])
}

func post(path, key string, body map[string]any) (int, map[string]any) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	return do(req)
}

func get(path string) (int, map[string]any) {
	req, _ := http.NewRequest("GET", baseURL+path, nil)
	return do(req)
}

func do(req *http.Request) (int, map[string]any) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail("%s %s: %v", req.Method, req.URL, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func fail(format string, args ...any) {
	fmt.Printf("error: "+format+"\n", args...)
	os.Exit(1)
}
