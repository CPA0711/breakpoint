package main

import (
	"crypto/tls" // <-- Penting buat SSL
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

type Result struct {
	C int
	N int
	RPS float64
	P50 int
	P95 int
	P99 int
	Err float64
}

func percentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func runTest(url string, c, n int, interval, warmer time.Duration) Result {
	client := &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	}
	var mu sync.Mutex
	latencies := make([]int, 0, n)
	errCount := 0

	for i := 0; i < int(warmer.Milliseconds())/100; i++ {
		client.Get(url)
		time.Sleep(100 * time.Millisecond)
	}

	jobs := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	bar := progressbar.NewOptions(n,
	progressbar.OptionSetWidth(8), // <-- BAR KECIL
	progressbar.OptionEnableColorCodes(true),
	progressbar.OptionSetPredictTime(false),
	progressbar.OptionSetDescription(fmt.Sprintf("C=%d Testing...", c)),
	progressbar.OptionShowCount(),
	progressbar.OptionSetTheme(progressbar.Theme{ // <-- Biar 100% nya gak nempel SUMMARY
			Saucer: "#",
			SaucerHead: "#",
			SaucerPadding: " ",
			BarStart: "|",
			BarEnd: "|",
	}),
	)

	var wg sync.WaitGroup
	sem := make(chan struct{}, c)
	startTime := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range jobs {
	<-ticker.C
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			start := time.Now()
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 BreakpointTester/1.0") // <-- Biar gak di-block
			resp, err := client.Do(req)
			dur := time.Since(start).Milliseconds()

			mu.Lock()
			defer mu.Unlock()
			if err!= nil || resp == nil || resp.StatusCode >= 400 {
				errCount++
			} else {
				latencies = append(latencies, int(dur))
			}
			if resp!= nil {
				resp.Body.Close()
			}
			bar.Add(1)
	}()
	}
	wg.Wait()
	bar.Finish() // <-- 1. Selesain barnya dulu
	fmt.Println() // <-- 2. Kasih enter 1x biar bar nya nongol beneran

	sort.Ints(latencies)
	p50 := percentile(latencies, 0.50)
	p95 := percentile(latencies, 0.95)
	p99 := percentile(latencies, 0.99)

	totalSeconds := time.Since(startTime).Seconds()
	rps := 0.0
	if totalSeconds > 0 && len(latencies) > 0 {
		rps = float64(len(latencies)) / totalSeconds
	}
	errPct := float64(errCount) / float64(n) * 100.0
	return Result{C: c, N: n, RPS: rps, P50: p50, P95: p95, P99: p99, Err: errPct}
}

func main() {
	url := flag.String("url", "https://google.com", "Target URL")
	n := flag.Int("n", 100, "Jumlah request per C")
	cMax := flag.Int("c", 10, "Max concurrency")
	step := flag.Duration("step", 10*time.Second, "Jeda antar C")
	warmer := flag.Duration("warmer", 3*time.Second, "Waktu warmup")
	interval := flag.Duration("interval", 100*time.Millisecond, "Jeda antar request")
	csvFile := flag.String("csv", "breakpoint.csv", "File output CSV")
	flag.Parse()

	file, _ := os.Create(*csvFile)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"C", "N", "RPS", "p50_ms", "p95_ms", "p99_ms", "Err_%"})

	fmt.Printf("🔥🔥🔥 CPA BREAKPOINT START 🔥🔥🔥 | Target: %s\n", *url)

	for c := 1; c <= *cMax; c++ {
		res := runTest(*url, c, *n, *interval, *warmer)
		writer.Write([]string{
			fmt.Sprintf("%d", res.C),
			fmt.Sprintf("%d", res.N),
			fmt.Sprintf("%.2f", res.RPS),
			fmt.Sprintf("%d", res.P50),
			fmt.Sprintf("%d", res.P95),
			fmt.Sprintf("%d", res.P99),
			fmt.Sprintf("%.1f", res.Err),
	})
		writer.Flush()

	// <-- TANPA \n di depan. Soalnya udah dikasih fmt.Println() di atas
		fmt.Printf("SUMMARY C=%d | RPS: %.2f | p50: %dms | p95: %dms | p99: %dms | Err: %.1f%%\n",
			res.C, res.RPS, res.P50, res.P95, res.P99, res.Err)

		if c < *cMax {
			time.Sleep(*step)
	}
	}
	fmt.Printf("\n🔥 BREAKPOINT SELESAI. CSV: %s\n", *csvFile)
}
