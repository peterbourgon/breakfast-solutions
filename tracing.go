package main

import (
	"context"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
)

func tracingAPIMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iw := &interceptingWriter{0, http.StatusOK, w}
		span, ctx := opentracing.StartSpanFromContext(r.Context(), "api_request")
		defer span.Finish()
		defer func(begin time.Time) {
			span.LogKV(
				"remote_addr", r.RemoteAddr,
				"method", r.Method,
				"url", r.URL,
				"content_length", r.ContentLength,
				"status_code", iw.code,
				"status_text", http.StatusText(iw.code),
				"response_size", iw.count,
				"took", time.Since(begin).String(),
				"sec", time.Since(begin).Seconds(),
			)
		}(time.Now())
		next.ServeHTTP(iw, r.WithContext(ctx))
	})
}

func tracingPreprocessMiddleware(next preprocessor) preprocessor {
	return func(ctx context.Context, region string) context.Context {
		span, ctx := opentracing.StartSpanFromContext(ctx, "preprocess")
		defer span.Finish()
		defer func(begin time.Time) {
			span.LogKV(
				"region", region,
				"took", time.Since(begin).String(),
				"sec", time.Since(begin).Seconds(),
			)
		}(time.Now())
		return next(ctx, region)
	}
}

type tracingRepoMiddleware struct {
	next repository
}

func (m tracingRepoMiddleware) getBreakfast(ctx context.Context, username string, breakfastID uint64) (b breakfast, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "db_request")
	defer span.Finish()
	defer func(begin time.Time) {
		span.LogKV(
			"method", "getBreakfast",
			"username", username,
			"breakfast_id", breakfastID,
			"took", time.Since(begin).String(),
			"sec", time.Since(begin).Seconds(),
			"success", err == nil,
			"returned_breakfast_id", b.ID,
			"err", err,
		)
	}(time.Now())
	return m.next.getBreakfast(ctx, username, breakfastID)
}

func (m tracingRepoMiddleware) getRandomBreakfast(ctx context.Context, username string) (b breakfast, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "db_request")
	defer span.Finish()
	defer func(begin time.Time) {
		span.LogKV(
			"method", "getRandomBreakfast",
			"username", username,
			"took", time.Since(begin).String(),
			"sec", time.Since(begin).Seconds(),
			"success", err == nil,
			"returned_breakfast_id", b.ID,
			"err", err,
		)
	}(time.Now())
	return m.next.getRandomBreakfast(ctx, username)
}

func tracingPostprocessMiddleware(next postprocessor) postprocessor {
	return func(ctx context.Context, username string, success bool) context.Context {
		span, ctx := opentracing.StartSpanFromContext(ctx, "postprocess")
		defer span.Finish()
		defer func(begin time.Time) {
			span.LogKV(
				"username", username,
				"success", success,
				"took", time.Since(begin).String(),
				"sec", time.Since(begin).Seconds(),
			)
		}(time.Now())
		return next(ctx, username, success)
	}
}
