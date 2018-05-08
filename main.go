package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/uber/jaeger-client-go"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	jaegermetrics "github.com/uber/jaeger-lib/metrics"
)

func main() {
	var (
		apiAddr    = flag.String("api", ":443", "API listen address")
		promAddr   = flag.String("prometheus", ":8081", "Prometheus listen address")
		jaegerAddr = flag.String("jaeger", "", "Jaeger host:port")
		oklogAddr  = flag.String("oklog", "", "OK Log host:port")
		cert       = flag.String("cert", "certs/server.crt", "TLS certificate")
		key        = flag.String("key", "certs/server.key", "TLS key")
		db         = flag.String("db", "breakfasts.json", "database file")
		images     = flag.String("images", "images/", "image dir")
		debug      = flag.Bool("debug", false, "print debug info")
	)
	flag.Parse()

	var console log.Logger
	{
		console = log.NewLogfmtLogger(os.Stderr)
		loglevel := level.AllowInfo()
		if *debug {
			loglevel = level.AllowDebug()
		}
		console = level.NewFilter(console, loglevel)
	}

	var structured log.Logger
	{
		if *oklogAddr != "" {
			conn, err := net.DialTimeout("tcp", *oklogAddr, time.Second)
			if err != nil {
				level.Error(console).Log("err", err)
				os.Exit(1)
			}
			defer conn.Close()
			structured = log.NewJSONLogger(conn)
			level.Info(console).Log("logging", "enabled", "oklog", *oklogAddr)
		} else {
			structured = log.NewNopLogger()
			level.Info(console).Log("logging", "disabled")
		}
	}

	var (
		duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "breakfast_solutions",
			Subsystem: "service",
			Name:      "request_duration_seconds",
			Help:      "Duration of each phase of a request in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"component", "operation", "success"})
	)

	{
		if *jaegerAddr != "" {
			transport, err := jaeger.NewUDPTransport(*jaegerAddr, 0)
			if err != nil {
				level.Error(console).Log("err", err)
				os.Exit(1)
			}
			cfg := jaegerconfig.Configuration{
				Sampler: &jaegerconfig.SamplerConfig{
					Type:  jaeger.SamplerTypeConst,
					Param: 1.0,
				},
			}
			closer, err := cfg.InitGlobalTracer(
				"breakfast_solutions",
				jaegerconfig.Logger(logAdapter{console}),
				jaegerconfig.Metrics(jaegermetrics.NullFactory),
				jaegerconfig.Reporter(jaeger.NewRemoteReporter(transport)),
			)
			if err != nil {
				level.Error(console).Log("err", err)
				os.Exit(1)
			}
			defer closer.Close()
			level.Info(console).Log("tracing", "enabled", "jaeger", *jaegerAddr)
		} else {
			level.Info(console).Log("tracing", "disabled")
		}
	}

	var pre preprocessor
	{
		pre = geoPreprocess
		pre = loggingPreprocessMiddleware(pre)
		pre = metricsPreprocessMiddleware(pre)
		pre = tracingPreprocessMiddleware(pre)
	}

	var repo repository
	{
		repo = mustNewRepository(*db)
		repo = loggingRepoMiddleware{repo}
		repo = metricsRepoMiddleware{repo}
		repo = tracingRepoMiddleware{repo}
	}

	var post postprocessor
	{
		post = basicPostprocess
		post = loggingPostprocessMiddleware(post)
		post = metricsPostprocessMiddleware(post)
		post = tracingPostprocessMiddleware(post)
	}

	var api http.Handler
	{
		api = newAPI(pre, repo, post, *images)
		api = hstsAPIMiddleware(api)
		api = loggingAPIMiddleware(api, structured)
		api = metricsAPIMiddleware(api, duration)
		api = tracingAPIMiddleware(api)
	}

	var g run.Group
	{
		server := &http.Server{Addr: *apiAddr, Handler: api}
		g.Add(func() error {
			level.Info(console).Log("api_addr", *apiAddr)
			return server.ListenAndServeTLS(*cert, *key)
		}, func(error) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			server.Shutdown(ctx)
		})
	}
	{
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		server := &http.Server{Addr: *promAddr, Handler: mux}
		g.Add(func() error {
			level.Info(console).Log("prometheus_addr", *promAddr)
			return server.ListenAndServe()
		}, func(error) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			server.Shutdown(ctx)
		})
	}
	{
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("received signal %s", sig)
			case <-ctx.Done():
				return ctx.Err()
			}
		}, func(error) {
			cancel()
		})
	}
	level.Info(console).Log("exit", g.Run())
}
