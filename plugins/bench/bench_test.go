package bench

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		sorted []time.Duration
		pct    int
		want   time.Duration
	}{
		{
			name:   "empty",
			sorted: nil,
			pct:    50,
			want:   0,
		},
		{
			name:   "single",
			sorted: []time.Duration{100 * time.Millisecond},
			pct:    50,
			want:   100 * time.Millisecond,
		},
		{
			name:   "P50 even",
			sorted: []time.Duration{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			pct:    50,
			want:   50, // ceil(50/100*10)-1 = 4 → sorted[4] = 50
		},
		{
			name:   "P99",
			sorted: []time.Duration{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			pct:    99,
			want:   100, // ceil(99/100*10)-1 = 9 → sorted[9] = 100
		},
		{
			name:   "P0",
			sorted: []time.Duration{10, 20, 30},
			pct:    0,
			want:   10, // ceil(0)-1 = -1, clamped to 0
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.sorted, tt.pct)
			if got != tt.want {
				t.Errorf("percentile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0ms"},
		{500 * time.Nanosecond, "0µs"},
		{100 * time.Microsecond, "100µs"},
		{5 * time.Millisecond, "5ms"},
		{1500 * time.Millisecond, "1.5s"},
		{2 * time.Second, "2s"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := roundDuration(tt.d)
			if got != tt.want {
				t.Errorf("roundDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestMin(t *testing.T) {
	if min(1, 2) != 1 {
		t.Error("min(1,2) should be 1")
	}
	if min(5, 3) != 3 {
		t.Error("min(5,3) should be 3")
	}
	if min(0, 0) != 0 {
		t.Error("min(0,0) should be 0")
	}
}

func TestRun_NoURL(t *testing.T) {
	_, err := Run(&BenchOptions{})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !strings.Contains(err.Error(), "URL is required") {
		t.Errorf("error = %q, want 'URL is required'", err.Error())
	}
}

func TestRun_RequestCount(t *testing.T) {
	// Create a simple test server
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	opts := &BenchOptions{
		URL:         server.URL,
		Method:      "GET",
		Concurrency: 2,
		Requests:    10,
		Timeout:     5 * time.Second,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.TotalRequests != 10 {
		t.Errorf("TotalRequests = %d, want 10", result.TotalRequests)
	}
	if result.SuccessRequests != 10 {
		t.Errorf("SuccessRequests = %d, want 10", result.SuccessRequests)
	}
	if result.FailedRequests != 0 {
		t.Errorf("FailedRequests = %d, want 0", result.FailedRequests)
	}
	if result.RequestsPerSecond <= 0 {
		t.Errorf("RequestsPerSecond = %f, want positive", result.RequestsPerSecond)
	}
	if result.MinLatency <= 0 {
		t.Error("MinLatency should be positive")
	}
	if result.MaxLatency < result.MinLatency {
		t.Error("MaxLatency should >= MinLatency")
	}
	if _, ok := result.StatusCodes[200]; !ok {
		t.Errorf("expected status 200 in StatusCodes, got %v", result.StatusCodes)
	}
}

func TestRun_WithPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-Custom") != "test-value" {
			t.Errorf("missing custom header")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	opts := &BenchOptions{
		URL:         server.URL,
		Method:      "POST",
		Headers:     map[string]string{"X-Custom": "test-value"},
		Body:        `{"hello":"world"}`,
		Concurrency: 1,
		Requests:    3,
		Timeout:     5 * time.Second,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", result.TotalRequests)
	}
	if _, ok := result.StatusCodes[201]; !ok {
		t.Errorf("expected status 201 in StatusCodes, got %v", result.StatusCodes)
	}
}

func TestRun_DurationBased(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := &BenchOptions{
		URL:         server.URL,
		Concurrency: 2,
		Duration:    500 * time.Millisecond,
		Timeout:     5 * time.Second,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		result, err := Run(opts)
		if err != nil {
			t.Errorf("Run() error: %v", err)
			return
		}
		if result.TotalRequests == 0 {
			t.Error("expected some requests in duration mode")
		}
		if result.SuccessRequests != result.TotalRequests {
			t.Errorf("expected all success, got %d/%d", result.SuccessRequests, result.TotalRequests)
		}
	}()

	select {
	case <-done:
		// ok
	case <-time.After(10 * time.Second):
		t.Fatal("Run() with Duration hung — potential bug in bench duration mode")
	}
}

func TestRun_ServerErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	opts := &BenchOptions{
		URL:         server.URL,
		Concurrency: 1,
		Requests:    5,
		Timeout:     5 * time.Second,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	// Server returns 500 but request itself succeeds (no network error)
	if result.TotalRequests != 5 {
		t.Errorf("TotalRequests = %d, want 5", result.TotalRequests)
	}
	if _, ok := result.StatusCodes[500]; !ok {
		t.Errorf("expected status 500 in StatusCodes, got %v", result.StatusCodes)
	}
}

func TestFormatResult(t *testing.T) {
	result := &Result{
		TotalRequests:     100,
		SuccessRequests:   95,
		FailedRequests:    5,
		TotalTime:         2 * time.Second,
		MinLatency:        10 * time.Millisecond,
		MaxLatency:        200 * time.Millisecond,
		AvgLatency:        50 * time.Millisecond,
		P50Latency:        45 * time.Millisecond,
		P90Latency:        80 * time.Millisecond,
		P95Latency:        120 * time.Millisecond,
		P99Latency:        180 * time.Millisecond,
		RequestsPerSecond: 50.0,
		StatusCodes:       map[int]int{200: 95, 500: 5},
		Errors:            []string{"connection refused"},
	}

	opts := &BenchOptions{
		URL:         "https://example.com/api",
		Method:      "GET",
		Concurrency: 10,
		Requests:    100,
	}

	output := FormatResult(result, opts)

	checks := []string{
		"example.com",
		"GET",
		"10",
		"95",
		"5",
		"50.00",
		"200",
		"500",
		"connection refused",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("FormatResult missing %q in output", check)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	result := &Result{
		TotalRequests:     50,
		SuccessRequests:   50,
		FailedRequests:    0,
		TotalTime:         1 * time.Second,
		MinLatency:        5 * time.Millisecond,
		AvgLatency:        20 * time.Millisecond,
		P50Latency:        18 * time.Millisecond,
		P90Latency:        35 * time.Millisecond,
		P95Latency:        40 * time.Millisecond,
		P99Latency:        45 * time.Millisecond,
		MaxLatency:        50 * time.Millisecond,
		RequestsPerSecond: 50.0,
		StatusCodes:       map[int]int{200: 50},
	}

	opts := &BenchOptions{
		URL:         "https://example.com",
		Method:      "GET",
		Concurrency: 5,
	}

	output := FormatJSON(result, opts)

	checks := []string{
		`"url": "https://example.com"`,
		`"method": "GET"`,
		`"total_requests": 50`,
		`"success_requests": 50`,
		`"qps": 50.00`,
		`"200": 50`,
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("FormatJSON missing %q in output:\n%s", check, output)
		}
	}
}

func TestRun_Defaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("default method should be GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Only URL and Requests set, everything else should default
	opts := &BenchOptions{
		URL:      server.URL,
		Requests: 5,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.TotalRequests != 5 {
		t.Errorf("TotalRequests = %d, want 5", result.TotalRequests)
	}
}
