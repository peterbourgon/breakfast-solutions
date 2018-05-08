package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
)

func loggingAPIMiddleware(next http.Handler, logger log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			iw  = &interceptingWriter{0, http.StatusOK, w}
			cl  = &contextLogger{}
			ctx = r.Context()
		)
		{
			ctx = context.WithValue(ctx, contextLoggerKey{}, cl)
		}
		cl.add(
			"http_req_remoteaddr", r.RemoteAddr,
			"http_req_method", r.Method,
			"http_req_url", r.URL.String(),
			"http_req_contentlength", r.ContentLength,
		)
		begin := time.Now()
		next.ServeHTTP(iw, r.WithContext(ctx))
		cl.add(
			"http_resp_statuscode", iw.code,
			"http_resp_statustext", http.StatusText(iw.code),
			"http_resp_size", iw.count,
			"http_resp_took", time.Since(begin).String(),
			"http_resp_sec", time.Since(begin).Seconds(),
		)
		logger.Log(cl.Keyvals...)
	})
}

func loggingPreprocessMiddleware(next preprocessor) preprocessor {
	return func(ctx context.Context, region string) context.Context {
		defer func(begin time.Time) {
			getContextLogger(ctx).add(
				"preprocess_region", region,
				"preprocess_took", time.Since(begin).String(),
				"preprocess_sec", time.Since(begin).Seconds(),
			)
		}(time.Now())
		return next(ctx, region)
	}
}

type loggingRepoMiddleware struct {
	next repository
}

func (m loggingRepoMiddleware) getBreakfast(ctx context.Context, username string, breakfastID uint64) (b breakfast, err error) {
	defer func(begin time.Time) {
		getContextLogger(ctx).add(
			"db_method", "getBreakfast",
			"db_username", username,
			"db_breakfast_id", breakfastID,
			"db_took", time.Since(begin).String(),
			"db_sec", time.Since(begin).Seconds(),
			"db_success", err == nil,
			"db_returned_breakfast_id", b.ID,
			"db_err", err,
		)
	}(time.Now())
	return m.next.getBreakfast(ctx, username, breakfastID)
}

func (m loggingRepoMiddleware) getRandomBreakfast(ctx context.Context, username string) (b breakfast, err error) {
	defer func(begin time.Time) {
		getContextLogger(ctx).add(
			"db_method", "getRandomBreakfast",
			"db_username", username,
			"db_took", time.Since(begin).String(),
			"db_sec", time.Since(begin).Seconds(),
			"db_success", err == nil,
			"db_returned_breakfast_id", b.ID,
			"db_err", err,
		)
	}(time.Now())
	return m.next.getRandomBreakfast(ctx, username)
}

func loggingPostprocessMiddleware(next postprocessor) postprocessor {
	return func(ctx context.Context, username string, success bool) context.Context {
		defer func(begin time.Time) {
			getContextLogger(ctx).add(
				"postprocess_username", username,
				"postprocess_success", fmt.Sprint(success),
				"postprocess_took", time.Since(begin).String(),
				"postprocess_sec", time.Since(begin).Seconds(),
			)
		}(time.Now())
		return next(ctx, username, success)
	}
}

//
//
//

type contextLoggerKey struct{}

type contextLogger struct{ Keyvals []interface{} }

func (l *contextLogger) add(keyvals ...interface{}) { l.Keyvals = append(l.Keyvals, keyvals...) }

func getContextLogger(ctx context.Context) *contextLogger {
	logger, ok := ctx.Value(contextLoggerKey{}).(*contextLogger)
	if !ok {
		panic("no context logger")
	}
	return logger
}
