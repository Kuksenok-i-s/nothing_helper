//go:build !systray

package tray

import (
	"context"

	"nothing_helper/internal/session"
)

func Run(ctx context.Context, s *session.Session, opts Options) {
	<-ctx.Done()
}
