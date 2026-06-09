package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"tws_manager/internal/bt"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
	"tws_manager/internal/trace"
)

// Runtime bundles shared application services after bootstrap.
type Runtime struct {
	Session   *session.Session
	Logger    *trace.Logger
	Config    Config
	TracePath string

	shutdownOnce sync.Once
}

// Bootstrap creates trace logger and session from validated config.
// Caller must call Runtime.Close when done.
func Bootstrap(ctx context.Context, cfg Config) (*Runtime, error) {
	tracePath := cfg.TracePath
	if tracePath == "" {
		tracePath = filepath.Join(cfg.CaptureDir, "session_"+time.Now().Format("2006-01-02_15-04-05")+".ndjson")
	}

	var logger *trace.Logger
	if tracePath != "" {
		var err error
		logger, err = trace.NewLogger(tracePath, cfg.LogRaw)
		if err != nil {
			return nil, fmt.Errorf("open trace log %q: %w", tracePath, err)
		}
	}

	sess := session.New(logger, cfg.AllowUnsafe, cfg.ProbeEnabled)
	sess.SetCaptureDir(cfg.CaptureDir)
	if cfg.ModelName != "" {
		model, ok := spp.ResolveModelInfo(cfg.ModelName)
		if !ok {
			if logger != nil {
				_ = logger.Close()
			}
			return nil, fmt.Errorf("unknown model %q", cfg.ModelName)
		}
		sess.SetModel(model)
	}
	queryEvery := cfg.QueryEvery
	if cfg.Notify && queryEvery <= 0 {
		queryEvery = 60 * time.Second
	}
	if queryEvery > 0 {
		sess.StartBatteryPolling(ctx, queryEvery)
	}

	return &Runtime{
		Session:   sess,
		Logger:    logger,
		Config:    cfg,
		TracePath: tracePath,
	}, nil
}

// WarmupPrivileges warms up the selected privilege backend when needed.
func WarmupPrivileges() (cached bool, err error) {
	return bt.WarmupPrivileges()
}

// Close shuts down logger and session. Prefer Shutdown for explicit lifecycle.
func (r *Runtime) Close() error {
	return r.Shutdown(context.Background())
}

// Warnf writes a warning to stderr (used by entrypoints for non-fatal issues).
func Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}
