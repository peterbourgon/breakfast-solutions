package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func hstsAPIMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

type interceptingWriter struct {
	count int
	code  int
	http.ResponseWriter
}

func (iw *interceptingWriter) WriteHeader(code int) {
	iw.code = code
	iw.ResponseWriter.WriteHeader(code)
}

func (iw *interceptingWriter) Write(p []byte) (int, error) {
	iw.count += len(p)
	return iw.ResponseWriter.Write(p)
}

func normalize(path string) string {
	switch {
	case path == "" || path == "/":
		return "/"
	default:
		return "/" + strings.FieldsFunc(path, func(r rune) bool { return r == '/' })[0]
	}
}

type logAdapter struct{ log.Logger }

func (a logAdapter) Error(msg string) {
	level.Error(a.Logger).Log("component", "Jaeger", "msg", msg)
}

func (a logAdapter) Infof(msg string, args ...interface{}) {
	level.Info(a.Logger).Log("component", "Jaeger", "msg", fmt.Sprintf(msg, args...))
}
