package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"math/rand"
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
	rand.Seed(time.Now().UnixNano()) // Wajib buat jitter

	url := flag.String("url", "", "URL target, wajib")
	n := flag.Int("n", 100, "Jumlah request per concurrency")
	maxC := flag.Int("max-c", 10, "Max concurrency. Jangan barbar, WAF sensitif")
	step := flag.Duration("step", 3*time.Second, "Jeda antar level C, kasih napas ke WAF")
	out := flag.String("out", "waf_human.csv", "File output CSV")
	to := flag.Duration("timeout", 20*time.Second, "Timeout per request, WAF suka lambat")
	flag.Parse()

	if *url == "" {
		fmt.Println("Usage: go run. -url https://target.com -n 100 -max-c 10")
		os.Exit(1)
	}

	client := &http.Client{Timeout: *to}

	var results []Result
	fmt.Printf("Starting HUMAN MODE Test -> %s\n", *url)
	fmt.Println("Rule Stop: Err% > 15.0% OR p99 > 2000ms") // Toleransi lebih longgar

	for c := 1; c <= *maxC; c++ {
		fmt.Printf(">> [C=%d] Testing %d requests...\n", c, *n)
		res := runTestHuman(*url, c, *n, client)
		results = append(results, res)
		fmt.Printf("<< [C=%d] DONE | RPS: %.2f | p50: %dms | p95: %dms | p99: %dms | Err: %.1f%%\n\n",
			res.C, res.RPS, res.P50, res.P95, res.P99, res.ErrPct)

		if c >= 2 {
			if shouldStop(res) {
				fmt.Printf("🛑 AUTO STOP: WAF Terpicu. Err=%.1f%% atau p99=%dms.\n", res.ErrPct, res.P99)
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

func shouldStop(r Result) bool {
	if r.ErrPct > 15.0 || r.P99 > 2000 { // Naikin threshold karena emang bakal lambat
		return true
	}
	return false
}

// runTestHuman: Versi yang pura-pura jadi manusia
func runTestHuman(url string, c, n int, client *http.Client) Result {
	var mu sync.Mutex
	var latencies []int64
	var errCount int64
	var okCount int64

	jobs := make(chan struct{}, n)
	for i := 0; i < n; i++ { jobs <- struct{}{} }
	close(jobs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < c; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for range jobs {
				select { case <-ctx.Done(): return default: }

				// 1. JITTER: Delay random 300ms - 800ms antar request
				time.Sleep(time.Duration(300+rand.Intn(500)) * time.Millisecond)

				t0 := time.Now()
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
				
				// 2. HEADER MANUSIA: Ini paling penting lawan WAF
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
				req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
				req.Header.Set("Accept-Language", "en-US,en;q=0.5")
				req.Header.Set("Referer", "https://www.google.com/") // WAF suka ini
				req.Header.Set("Connection", "keep-alive")
				
				resp, err := client.Do(req)
				dt := time.Since(t0).Milliseconds()

				if err!= nil || (resp!= nil && resp.StatusCode >= 400) {
					atomic.AddInt64(&errCount, 1)
					if resp!= nil { resp.Body.Close() }
					continue
				}
				resp.Body.Close()
				atomic.AddInt64(&okCount, 1)

				mu.Lock()
				latencies = append(latencies, dt)
				mu.Unlock()
			}
	}(i)
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
	if total > 0 { errPct = float64(errCount) / float64(total) * 100 }
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
