package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

func percentile(data []int64, p float64) float64 {
	if len(data) == 0 { return 0 }
	sort.Slice(data, func(i, j int) bool { return data[i] < data[j] })
	k := p * float64(len(data)-1)
	f := math.Floor(k)
	i := int(f)
	if i+1 < len(data) { return float64(data[i]) + (k-f)*float64(data[i+1]-data[i]) }
	return float64(data[i])
}

func main() {
	url := flag.String("url", "http://localhost:3000", "Target URL")
	total := flag.Int("n", 100, "Total requests per step")
	cMax := flag.Int("c", 10, "Max concurrency")
	interval := flag.Duration("interval", 200*time.Millisecond, "Jeda antar request")
	csvFile := flag.String("csv", "/sdcard/breakpoint.csv", "CSV")
	warmer := flag.Duration("warmer", 5*time.Second, "Durasi warmer")
	stepTime := flag.Duration("step", 30*time.Second, "Lama per step")
	flag.Parse()

	file, _ := os.Create(*csvFile)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"timestamp", "status_code", "duration_ms", "success", "concurrency"})

	client := &http.Client{Timeout: 10 * time.Second}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println(color.YellowString("Target: %s | Max C: %d | Step: %s", *url, *cMax, *stepTime))

	for c := 1; c <= *cMax; c++ {
		select {
		case <-ctx.Done():
			fmt.Println(color.RedString("\nDibatalkan"))
			return
		default:
	}

		fmt.Println(color.CyanString("\n===== C=%d =====", c))
		var mu sync.Mutex
		var durations []int64
		var statusCount = make(map[int]int)
		var successCount, totalDone int
		var active int32
		var isWarming = true
		var startTime time.Time

	bar := progressbar.NewOptions(*total,
			progressbar.OptionSetDescription(fmt.Sprintf("C=%d Warming...", c)),
			progressbar.OptionShowCount(), progressbar.OptionSetWidth(20))

		sem := make(chan struct{}, c)
		var wg sync.WaitGroup
		ticker := time.NewTicker(*interval)
		stepTimer := time.NewTimer(*stepTime)
		defer ticker.Stop()
		defer stepTimer.Stop()

		if *warmer > 0 {
			time.AfterFunc(*warmer, func() {
				mu.Lock()
				isWarming = false
				startTime = time.Now()
				mu.Unlock()
				bar.Describe(fmt.Sprintf("C=%d Testing...", c))
			})
	} else {
			isWarming = false
			startTime = time.Now()
	}

		reqCount := 0
	STEP_LOOP:
		for {
			select {
			case <-ctx.Done(): break STEP_LOOP
			case <-stepTimer.C: break STEP_LOOP
			case <-ticker.C:
				if reqCount >= *total { continue }
				reqCount++
				wg.Add(1)
				sem <- struct{}{}
				go func() {
					defer wg.Done()
					defer func() { <-sem }()
					atomic.AddInt32(&active, 1)
					defer atomic.AddInt32(&active, -1)

					start := time.Now()
					req, _ := http.NewRequestWithContext(ctx, "GET", *url, nil)
					resp, err := client.Do(req)
					dur := time.Since(start).Milliseconds()
					code, ok := 0, false
					if err == nil {
						code = resp.StatusCode
						ok = resp.StatusCode >= 200 && resp.StatusCode < 300
						resp.Body.Close()
					}

					mu.Lock()
					if!isWarming {
						durations = append(durations, dur)
						statusCount[code]++
						if ok { successCount++ }
						totalDone++
						writer.Write([]string{time.Now().Format("15:04:05"), fmt.Sprintf("%d", code), fmt.Sprintf("%d", dur), fmt.Sprintf("%t", ok), fmt.Sprintf("%d", c)})
						bar.Add(1)
					}
					mu.Unlock()
				}()
			}
	}
		wg.Wait()
	bar.Finish()

		if totalDone > 0 {
			elapsed := time.Since(startTime).Seconds()
			fmt.Printf("%s C=%d | RPS: %.2f | p50: %.0fms | p95: %.0fms | p99: %.0fms | Err: %.1f%%\n",
				color.GreenString("SUMMARY"),
				c, float64(totalDone)/elapsed, percentile(durations, 0.50), percentile(durations, 0.95), percentile(durations, 0.99),
				100*float64(totalDone-successCount)/float64(totalDone))
	}
	}
	fmt.Println(color.GreenString("\n🔥 BREAKPOINT SELESAI. CSV: %s", *csvFile))
}
