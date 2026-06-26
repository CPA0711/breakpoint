package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
	"github.com/schollz/progressbar/v3"
)

type Result struct {
	C int `csv:"concurrency"`
	RPS float64 `csv:"rps"`
	P50 int `csv:"p50_ms"`
	P95 int `csv:"p95_ms"`
	P99 int `csv:"p99_ms"`
	ErrPct float64 `csv:"error_pct"`
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0",
}

func main() {
	url := flag.String("url", "https://www.alibaba.com", "Target URL") // Default ganti ke Alibaba
	maxC := flag.Int("c", 20, "Max Concurrency")
	n := flag.Int("n", 20, "Requests per concurrency level")
	interval := flag.Duration("interval", 50*time.Millisecond, "Delay between requests")
	step := flag.Duration("step", 10*time.Second, "Wait time between levels")
	out := flag.String("out", "breakpoint.csv", "Output CSV file")
	tolerance := flag.Float64("tol", 0.05, "RPS tolerance for flat detection. 0.05 = 5%") // <-- FITUR BARU
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	fmt.Println("🚀 STARTING CPA BREAKPOINT...")
	fmt.Printf("Target: %s | -c=%d -n=%d | AutoStop Tol: %.0f%%\n", *url, *maxC, *n, *tolerance*100)

	client := &http.Client{Timeout: 10 * time.Second}
	var results []Result

	for c := 1; c <= *maxC; c++ {
		fmt.Printf(">> [C=%d] Testing %d requests...\n", c, *n)
		res := runTest(*url, c, *n, *interval, client)
		results = append(results, res)
		fmt.Printf("<< [C=%d] DONE | RPS: %.2f | p50: %dms | p95: %dms | p99: %dms | Err: %.1f%%\n\n",
			res.C, res.RPS, res.P50, res.P95, res.P99, res.ErrPct)

		// <-- LOGIKA AUTO STOP DIMULAI DARI SINI
		if c >= 4 { // Minimal tes sampe C=4 biar ada data
			if isFlat(results, int(*tolerance*100)) { // Cek 3 data terakhir
				fmt.Printf("🛑 AUTO STOP: RPS flat 3x berturut-turut di ~%.2f RPS. Breakpoint ketemu.\n", res.RPS)
				break
			}
	}

		if c < *maxC {
			time.Sleep(*step)
	}
	}

	writeCSV(*out, results)
	fmt.Printf("🔥 SELESAI. CSV: %s | Total Level Diuji: %d\n", *out, len(results))
}

func runTest(url string, c, n int, interval time.Duration, client *http.Client) Result {
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, c)
	bar := progressbar.NewOptions(n,
	progressbar.OptionSetWidth(15),
	progressbar.OptionShowCount(),
	progressbar.OptionSetPredictTime(false),
	progressbar.OptionClearOnFinish(),
	)

	var latencies []int
	errCount := 0
	startTime := time.Now()

	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			start := time.Now()
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
			req.Header.Set("Referer", "https://www.google.com/")
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9")
			req.Header.Set("Accept-Language", "en-US,en;q=0.5")
			req.Header.Set("Connection", "keep-alive")

			resp, err := client.Do(req)
			dur := time.Since(start).Milliseconds()

			mu.Lock()
			if err!= nil || resp == nil || resp.StatusCode >= 400 {
				errCount++
			} else {
				latencies = append(latencies, int(dur))
			}
			if resp!= nil {
				resp.Body.Close()
			}
			mu.Unlock()
			bar.Add(1)
	}()
		time.Sleep(interval)
	}
	wg.Wait()
	totalTime := time.Since(startTime).Seconds()

	sort.Ints(latencies)
	p50, p95, p99 := percentile(latencies, 50), percentile(latencies, 95), percentile(latencies, 99)
	rps := float64(len(latencies)) / totalTime
	errPct := float64(errCount) / float64(n) * 100
	return Result{c, rps, p50, p95, p99, errPct}
}

func percentile(data []int, p int) int {
	if len(data) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100.0*float64(len(data)))) - 1
	if idx < 0 {
		idx = 0
	}
	return data[idx]
}

func writeCSV(filename string, results []Result) {
	f, _ := os.Create(filename)
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write([]string{"concurrency", "rps", "p50_ms", "p95_ms", "p99_ms", "error_pct"})
	for _, r := range results {
		w.Write([]string{
			fmt.Sprint(r.C),
			fmt.Sprintf("%.2f", r.RPS),
			fmt.Sprint(r.P50),
			fmt.Sprint(r.P95),
			fmt.Sprint(r.P99),
			fmt.Sprintf("%.1f", r.ErrPct),
	})
	}
}

// <-- FUNGSI BARU: Cek Flat 3x
func isFlat(results []Result, tolPct int) bool {
	if len(results) < 3 {
		return false
	}
	last3 := results[len(results)-3:]
	base := last3[0].RPS
	for _, r := range last3[1:] {
		diff := math.Abs(r.RPS - base) / base
		if diff > float64(tolPct)/100.0 { // Toleransi 5%
			return false
	}
	}
	return true
}
}
}
