package app

import (
	"context"
	"testing"
	"time"
)

func TestStartPprofNoOpWhenAddrEmpty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		StartPprof(ctx, "")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("StartPprof with empty addr should return immediately")
	}
}
