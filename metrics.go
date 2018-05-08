package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func metricsAPIMiddleware(next http.Handler, duration *prometheus.HistogramVec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			iw  = &interceptingWriter{0, http.StatusOK, w}
			ctx = context.WithValue(r.Context(), contextHistogramKey{}, duration)
		)
		defer func(begin time.Time) {
			duration.WithLabelValues(
				"API", normalize(r.URL.Path), fmt.Sprint(iw.code == 200),
			).Observe(time.Since(begin).Seconds())
		}(time.Now())
		next.ServeHTTP(iw, r.WithContext(ctx))
	})
}

func metricsPreprocessMiddleware(next preprocessor) preprocessor {
	return func(ctx context.Context, region string) context.Context {
		defer func(begin time.Time) {
			getContextHistogram(ctx).WithLabelValues(
				"preprocessor", "preprocess", "true",
			).Observe(time.Since(begin).Seconds())
		}(time.Now())
		return next(ctx, region)
	}
}

type metricsRepoMiddleware struct {
	next repository
}

func (m metricsRepoMiddleware) getBreakfast(ctx context.Context, username string, breakfastID uint64) (b breakfast, err error) {
	defer func(begin time.Time) {
		getContextHistogram(ctx).WithLabelValues(
			"DB", "getBreakfast", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return m.next.getBreakfast(ctx, username, breakfastID)
}

func (m metricsRepoMiddleware) getRandomBreakfast(ctx context.Context, username string) (b breakfast, err error) {
	defer func(begin time.Time) {
		getContextHistogram(ctx).WithLabelValues(
			"DB", "getRandomBreakfast", fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())
	return m.next.getRandomBreakfast(ctx, username)
}

func metricsPostprocessMiddleware(next postprocessor) postprocessor {
	return func(ctx context.Context, username string, success bool) context.Context {
		defer func(begin time.Time) {
			getContextHistogram(ctx).WithLabelValues(
				"postprocessor", "postprocess", fmt.Sprint(success),
			).Observe(time.Since(begin).Seconds())
		}(time.Now())
		return next(ctx, username, success)
	}
}

//
//
//

type contextHistogramKey struct{}

func getContextHistogram(ctx context.Context) *prometheus.HistogramVec {
	histogram, ok := ctx.Value(contextHistogramKey{}).(*prometheus.HistogramVec)
	if !ok {
		panic("no context histogram")
	}
	return histogram
}
