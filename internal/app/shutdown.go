package app

import (
	"context"
	"sync"
	"time"
)

// Shutdown closes the RFCOMM session and flushes the trace log.
// It is safe to call more than once.
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var err error
	r.shutdownOnce.Do(func() {
		err = r.shutdownOnceBody(ctx)
	})
	return err
}

func (r *Runtime) shutdownOnceBody(ctx context.Context) error {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	var err error
	if r.Session != nil {
		if closeErr := r.Session.Close(); closeErr != nil {
			err = closeErr
		}
	}
	return firstError(err, r.closeBootstrapLogger())
}

func (r *Runtime) closeBootstrapLogger() error {
	if r.Logger == nil {
		return nil
	}
	err := r.Logger.Close()
	r.Logger = nil
	return err
}

func firstError(primary, secondary error) error {
	if primary != nil {
		return primary
	}
	return secondary
}

// Run bootstraps the runtime, invokes fn, and always shuts down on return or ctx cancel.
func Run(ctx context.Context, cfg Config, fn func(context.Context, *Runtime) error) error {
	rt, err := Bootstrap(ctx, cfg)
	if err != nil {
		return err
	}

	// WithoutCancel keeps request-scoped values while allowing shutdown after
	// the run context is cancelled. Keep this short: Session.Close already
	// time-bounds RFCOMM teardown; we only need a moment for the logger flush.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer shutdownCancel()

	var once sync.Once
	shutdown := func() {
		once.Do(func() {
			_ = rt.Shutdown(shutdownCtx)
			shutdownCancel()
		})
	}
	defer shutdown()

	go func() {
		<-ctx.Done()
		shutdown()
	}()

	go StartPprof(ctx, cfg.PprofAddr)

	if err := fn(ctx, rt); err != nil {
		return err
	}
	return nil
}
