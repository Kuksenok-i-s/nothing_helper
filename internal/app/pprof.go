package app

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
)

// StartPprof serves Go runtime profiles on addr (e.g. 127.0.0.1:6060).
// Endpoints: /debug/pprof/profile, /debug/pprof/heap, /debug/pprof/goroutine.
// No-op when addr is empty.
func StartPprof(ctx context.Context, addr string) {
	if addr == "" {
		return
	}
	srv := &http.Server{Addr: addr, Handler: http.DefaultServeMux}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	fmt.Fprintf(os.Stderr, "pprof: http://%s/debug/pprof/ (profile: curl -o cpu.prof 'http://%s/debug/pprof/profile?seconds=30')\n", addr, addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		Warnf("pprof server: %v", err)
	}
}
