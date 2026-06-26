package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	C int
	RPS float64
	P50 int64
	P95 int
	P99 int
	ErrPct float64
}

func main() {
	url := flag.String("url", "", "URL target, wajib")
	n := flag.Int("n", 100, "Jumlah request per concurrency")
	maxC := flag.Int("max-c", 20, "Max concurrency")
	step := flag.Duration("step", 2*time.Second, "Jeda antar level C")
	interval := flag.Duration("interval", 0, "Jeda antar request per worker, 0 = secepatnya")
	out := flag.String("out", "waf_test.csv", "File output CSV")
	to := flag.Duration("timeout", 15*time.Second, "Timeout per request")
	flag.Parse()

	if *url == "" {
		fmt.Println("Usage: go run. -url https://target.com -n 100 -max-c 20")
		os.Exit(1)
	}

	client := &http.Client{
	Timeout: *to,
	Transport: &http.Transport{
			MaxIdleConns: *maxC,
			MaxIdleConnsPerHost: *maxC,
			DisableKeepAlives: false,
	},
	}

	var results []Result
	fmt.Printf("Starting WAF Test -> %s\n", *url)
	fmt.Println("Rule Stop: Err% > 10.0% OR p99 > 900ms")

	for c := 1; c <= *maxC; c++ {
		fmt.Printf(">> [C=%d] Testing %d requests...\n", c, *n)
		res := runTest(*url, c, *n, *interval, client)
		results = append(results, res)
		fmt.Printf("<< [C=%d] DONE | RPS: %.2f | p50: %dms | p95: %dms | p99: %dms | Err: %.1f%%\n\n",
			res.C, res.RPS, res.P50, res.P95, res.P99, res.ErrPct)

	// LOGIKA AUTO STOP V2: UNTUK WAF CHINA
		if c >= 2 { // Skip C=1 karena WAF biasanya sengaja kill C=1
			if shouldStop(res) {
				fmt.Printf("🛑 AUTO STOP: WAF Terpicu. Err=%.1f%% atau p99=%dms. IP lu udah di greylist.\n", res.ErrPct, res.P99)
				break
			}
	}

		if c < *maxC {
			time.Sleep(*step)
	}
	}

	writeCSV(*out, results)
	fmt.Printf("Selesai. Hasil di %s\n", *out)
}

// shouldStop: Kunci buat lawan WAF. Kalo udah kena, stop.
func shouldStop(r Result) bool {
	if r.ErrPct > 10.0 || r.P99 > 900 {
		return true
	}
	return false
}

func runTest(url string, c, n int, interval time.Duration, client *http.Client) Result {
	var mu sync.Mutex
	var latencies []int64
	var errCount int64
	var okCount int64

	jobs := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < c; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				t0 := time.Now()
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
				req.Header.Set("User-Agent", "Mozilla/5.0 LoadTest/1.0") // WAF benci UA default Go
				resp, err := client.Do(req)
				dt := time.Since(t0).Milliseconds()

				if err!= nil || (resp!= nil && resp.StatusCode >= 400) {
					atomic.AddInt64(&errCount, 1)
					if resp!= nil {
						resp.Body.Close()
					}
					continue
				}
				resp.Body.Close()
				atomic.AddInt64(&okCount, 1)

				mu.Lock()
				latencies = append(latencies, dt)
				mu.Unlock()

				if interval > 0 {
					time.Sleep(interval)
				}
			}
	}()
	}
	wg.Wait()
	totalTime := time.Since(start).Seconds()

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	
	p50, p95, p99 := 0, 0, 0
	if len(latencies) > 0 {
		p50 = int(latencies[int(math.Floor(float64(len(latencies))*0.5))])
		p95 = int(latencies[int(math.Floor(float64(len(latencies))*0.95))])
		p99 = int(latencies[int(math.Floor(float64(len(latencies))*0.99))])
	}

	total := okCount + errCount
	errPct := 0.0
	if total > 0 {
		errPct = float64(errCount) / float64(total) * 100
	}
	rps := float64(okCount) / totalTime

	return Result{c, rps, int64(p50), p95, p99, errPct}
}

func writeCSV(path string, rs []Result) {
	f, _ := os.Create(path)
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write([]string{"concurrency", "rps", "p50_ms", "p95_ms", "p99_ms", "error_pct"})
	for _, r := range rs {
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
