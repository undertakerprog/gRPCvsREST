package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gRPCvsREST/api/proto/todopb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

const (
	warmupRequests = 1000
	requestTimeout = 5 * time.Second
)

type result struct {
	latency time.Duration
	bytes   int
	err     error
}

func main() {
	var (
		mode      = flag.String("mode", "rest", "rest|grpc")
		baseURL   = flag.String("base", "http://localhost:8080", "REST base URL")
		grpcAddr  = flag.String("grpc", "localhost:9090", "gRPC address")
		totalReq  = flag.Int("n", 20000, "number of requests")
		conc      = flag.Int("c", 50, "concurrency")
		payloadKB = flag.Int("payload_kb", 32, "payload size in KB")
		limit     = flag.Int("limit", 100, "ListTodos limit")
	)
	flag.Parse()

	if *totalReq <= 0 || *conc <= 0 {
		log.Fatal("n and c must be > 0")
	}
	if *payloadKB < 0 || *limit < 0 {
		log.Fatal("payload_kb and limit must be >= 0")
	}

	var doRequest func() (int, error)
	var cleanup func()

	switch strings.ToLower(*mode) {
	case "rest":
		urlStr, err := buildRestURL(*baseURL, *limit, *payloadKB)
		if err != nil {
			log.Fatalf("invalid base url: %v", err)
		}
		client := &http.Client{}
		doRequest = func() (int, error) {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
			if err != nil {
				return 0, err
			}

			resp, err := client.Do(req)
			if err != nil {
				return 0, err
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				return 0, fmt.Errorf("unexpected status: %s", resp.Status)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return 0, err
			}
			return len(body), nil
		}
	case "grpc":
		conn, err := grpc.NewClient(*grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("grpc dial error: %v", err)
		}
		client := todopb.NewTodoServiceClient(conn)
		doRequest = func() (int, error) {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			resp, err := client.ListTodos(ctx, &todopb.ListTodosRequest{
				Limit:     int32(*limit),
				Offset:    0,
				PayloadKb: int32(*payloadKB),
			})
			if err != nil {
				return 0, err
			}
			data, err := proto.Marshal(resp)
			if err != nil {
				return 0, err
			}
			return len(data), nil
		}
		cleanup = func() {
			_ = conn.Close()
		}
	default:
		log.Fatal("mode must be rest or grpc")
	}

	if cleanup != nil {
		defer cleanup()
	}

	warmupN := warmupRequests
	if *totalReq < warmupN {
		warmupN = *totalReq
	}

	fmt.Printf("warmup: %d requests\n", warmupN)
	_ = runBatch(warmupN, *conc, doRequest)

	fmt.Printf("run: mode=%s n=%d c=%d payload_kb=%d limit=%d\n", *mode, *totalReq, *conc, *payloadKB, *limit)
	metrics := runBatch(*totalReq, *conc, doRequest)
	printMetrics(metrics)
}

type metrics struct {
	latencies []time.Duration
	bytes     int64
	errors    int
	duration  time.Duration
}

func runBatch(n, c int, do func() (int, error)) metrics {
	jobs := make(chan struct{}, n)
	results := make(chan result, n)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for range jobs {
			start := time.Now()
			b, err := do()
			lat := time.Since(start)
			results <- result{latency: lat, bytes: b, err: err}
		}
	}

	for i := 0; i < c; i++ {
		wg.Add(1)
		go worker()
	}

	start := time.Now()
	for i := 0; i < n; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	out := metrics{
		latencies: make([]time.Duration, 0, n),
	}
	for res := range results {
		if res.err != nil {
			out.errors++
			continue
		}
		out.latencies = append(out.latencies, res.latency)
		out.bytes += int64(res.bytes)
	}
	out.duration = time.Since(start)
	return out
}

func printMetrics(m metrics) {
	success := len(m.latencies)
	if success == 0 {
		fmt.Printf("errors=%d, no successful requests\n", m.errors)
		os.Exit(1)
	}

	sort.Slice(m.latencies, func(i, j int) bool { return m.latencies[i] < m.latencies[j] })
	p50 := percentile(m.latencies, 0.50)
	p95 := percentile(m.latencies, 0.95)
	rps := float64(success) / m.duration.Seconds()
	avgBytes := float64(m.bytes) / float64(success)

	fmt.Printf("requests=%d success=%d errors=%d\n", success+m.errors, success, m.errors)
	fmt.Printf("latency_ms p50=%.2f p95=%.2f\n", p50, p95)
	fmt.Printf("rps=%.2f duration=%s\n", rps, m.duration.Round(time.Millisecond))
	fmt.Printf("bytes_total=%d bytes_avg=%.2f\n", m.bytes, avgBytes)
}

func percentile(latencies []time.Duration, p float64) float64 {
	if len(latencies) == 0 {
		return 0
	}
	if p <= 0 {
		return float64(latencies[0]) / float64(time.Millisecond)
	}
	if p >= 1 {
		return float64(latencies[len(latencies)-1]) / float64(time.Millisecond)
	}
	idx := int(math.Ceil(p*float64(len(latencies)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(latencies) {
		idx = len(latencies) - 1
	}
	return float64(latencies[idx]) / float64(time.Millisecond)
}

func buildRestURL(base string, limit, payloadKB int) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" {
		return "", errors.New("base must include scheme")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/todos"
	query := parsed.Query()
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("offset", "0")
	query.Set("payload_kb", fmt.Sprintf("%d", payloadKB))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
