package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	C int
	Proxy string
	RPS float64
	P50 int64
	P95 int
	P99 int
	ErrPct float64
}

func main() {
	rand.Seed(time.Now().UnixNano())

	urlTarget := flag.String("url", "", "URL target, wajib")
	proxyFile := flag.String("proxies", "proxies.txt", "File txt isi list proxy 1 baris 1")
	maxC := flag.Int("max-c", 4, "Max concurrency PER PROXY. Jangan >4")
	n := flag.Int("n", 50, "Jumlah request per proxy")
	out := flag.String("out", "waf_proxy.csv", "File output CSV")
	to := flag.Duration("timeout", 20*time.Second, "Timeout per request")
	flag.Parse()

	if *urlTarget == "" {
		fmt.Println("Usage: go run. -url https://target.com -proxies proxies.txt")
		os.Exit(1)
	}

	proxies := readProxies(*proxyFile)
	if len(proxies) == 0 {
		fmt.Println("File proxies.txt kosong")
		os.Exit(1)
	}

	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	fmt.Printf("Starting PROXY ROTATOR Test -> %s | Proxies: %d | C/Proxy: %d\n", *urlTarget, len(proxies), *maxC)

	for _, p := range proxies {
		wg.Add(1)
		go func(proxyURL string) {
			defer wg.Done()
			client := newClient(proxyURL, *to)
			res := runTestHuman(*urlTarget, *maxC, *n, client, proxyURL)
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
			fmt.Printf("<< [Proxy %s] DONE | RPS: %.2f | p99: %dms | Err: %.1f%%\n", maskProxy(proxyURL), res.RPS, res.P99, res.ErrPct)
	}(p)
		time.Sleep(1 * time.Second) // Jeda biar gak start bareng
	}
	wg.Wait()
	writeCSV(*out, results)
	fmt.Printf("\nSelesai. Total Proxy: %d. Hasil di %s\n", len(proxies), *out)
}

func newClient(proxyStr string, to time.Duration) *http.Client {
	proxyURL, err := url.Parse(proxyStr)
	if err!= nil {
		return &http.Client{Timeout: to}
	}
	transport := &http.Transport{
	Proxy: http.ProxyURL(proxyURL),
	}
	return &http.Client{Timeout: to, Transport: transport}
}

func runTestHuman(url string, c, n int, client *http.Client, proxy string) Result {
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
				time.Sleep(time.Duration(300+rand.Intn(500)) * time.Millisecond) // Jitter

				t0 := time.Now()
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")
				req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")
				req.Header.Set("Referer", "https://www.google.com/")

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

	return Result{c, proxy, rps, int64(p50), p95, p99, errPct}
}

func readProxies(path string) []string {
	f, err := os.Open(path)
	if err!= nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s!= "" {
			out = append(out, s)
	}
	}
	return out
}

func writeCSV(path string, rs []Result) {
	f, _ := os.Create(path)
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write([]string{"concurrency_per_proxy", "proxy", "rps", "p50_ms", "p95_ms", "p99_ms", "error_pct"})
	for _, r := range rs {
		w.Write([]string{
			fmt.Sprint(r.C),
			maskProxy(r.Proxy),
			fmt.Sprintf("%.2f", r.RPS),
			fmt.Sprint(r.P50),
			fmt.Sprint(r.P95),
			fmt.Sprint(r.P99),
			fmt.Sprintf("%.1f", r.ErrPct),
	})
	}
}

func maskProxy(s string) string {
	// biar log gak kebuka full
	parts := strings.Split(s, "@")
	if len(parts) == 2 {
		return "***@" + parts[1]
	}
	return s
}
