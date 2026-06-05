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
		if ctx != nil {
			select {
			case <-ctx.Done():
				err = ctx.Err()
			default:
			}
		}
		if r.Session != nil {
			if closeErr := r.Session.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
		if r.Logger != nil {
			if logErr := r.Logger.Close(); logErr != nil && err == nil {
				err = logErr
			}
			r.Logger = nil
		}
	})
	return err
}

// Run bootstraps the runtime, invokes fn, and always shuts down on return or ctx cancel.
func Run(ctx context.Context, cfg Config, fn func(context.Context, *Runtime) error) error {
	rt, err := Bootstrap(ctx, cfg)
	if err != nil {
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	if err := fn(ctx, rt); err != nil {
		return err
	}
	return nil
}
