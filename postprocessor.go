package main

import (
	"context"
	"math/rand"
	"time"
)

type postprocessor func(ctx context.Context, username string, success bool) context.Context

func basicPostprocess(ctx context.Context, username string, success bool) context.Context {
	d := 1
	if !success {
		d *= 50
	}
	time.Sleep(time.Duration(d+rand.Intn(d)) * time.Millisecond)
	return ctx
}
