//go:build !systray

package tray

import (
	"context"

	"tws_manager/internal/session"
)

type Options struct {
	AppName      string
	OnReconnect  func()
	OnShowWindow func()
	OnQuit       func()
}

func Run(ctx context.Context, s *session.Session, opts Options) {
	<-ctx.Done()
}
