package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nothing_helper/internal/bt"
	"nothing_helper/internal/session"
	"nothing_helper/internal/spp"
	"nothing_helper/internal/trace"
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

	logger, err := openBootstrapLogger(tracePath, cfg.LogRaw)
	if err != nil {
		return nil, err
	}

	sess := session.New(logger, cfg.AllowUnsafe, cfg.ProbeEnabled)
	if cfg.LogRaw {
		sess.SetCaptureDir(cfg.CaptureDir)
	}
	if err := applyBootstrapModel(sess, logger, cfg.ModelName); err != nil {
		return nil, err
	}
	startBootstrapPolling(ctx, sess, cfg)

	return &Runtime{
		Session:   sess,
		Logger:    logger,
		Config:    cfg,
		TracePath: tracePath,
	}, nil
}

func openBootstrapLogger(tracePath string, logRaw bool) (*trace.Logger, error) {
	if tracePath == "" {
		return nil, nil
	}
	logger, err := trace.NewLogger(tracePath, logRaw)
	if err != nil {
		return nil, fmt.Errorf("open trace log %q: %w", tracePath, err)
	}
	return logger, nil
}

func applyBootstrapModel(sess *session.Session, logger *trace.Logger, modelName string) error {
	if modelName == "" {
		return nil
	}
	model, ok := spp.ResolveModelInfo(modelName)
	if !ok {
		if logger != nil {
			_ = logger.Close()
		}
		return fmt.Errorf("unknown model %q", modelName)
	}
	sess.SetModel(model)
	return nil
}

func startBootstrapPolling(ctx context.Context, sess *session.Session, cfg Config) {
	queryEvery := cfg.QueryEvery
	if cfg.Notify && queryEvery <= 0 {
		queryEvery = 60 * time.Second
	}
	if queryEvery > 0 {
		sess.StartBatteryPolling(ctx, queryEvery)
	}
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
