//go:build !systray

package tray

import (
	"context"

	"tws_manager/internal/session"
)

func Run(ctx context.Context, s *session.Session, opts Options) {
	<-ctx.Done()
}
