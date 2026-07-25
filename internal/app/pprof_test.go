package app

import (
	"context"
	"testing"
	"time"
)

func TestStartPprofNoOpWhenAddrEmpty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	StartPprof(ctx, "")
}

func TestStartPprofListensUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	StartPprof(ctx, "127.0.0.1:0")
}
