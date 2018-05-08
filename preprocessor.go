package main

import (
	"context"
	"strings"
	"time"
)

type preprocessor func(ctx context.Context, originIP string) context.Context

func geoPreprocess(ctx context.Context, region string) context.Context {
	var delay time.Duration
	switch strings.ToLower(region) {
	case "au":
		delay = 100 * time.Millisecond
	default:
		delay = 1 * time.Millisecond
	}
	time.Sleep(delay)
	return ctx
}
