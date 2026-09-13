//go:build !gio

package companion

import (
	"context"
	"fmt"
)

func Available() bool { return false }

func Run(context.Context, Options) error {
	return fmt.Errorf("GUI is not included in this build; use make build or go run -tags gio ./cmd/nothing_helper")
}
