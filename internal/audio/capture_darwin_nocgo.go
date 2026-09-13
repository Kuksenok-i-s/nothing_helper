//go:build darwin && !cgo

package audio

import (
	"context"
	"fmt"
)

func HasActiveCaptureForMAC(context.Context, string) (bool, error) {
	return false, fmt.Errorf("проверка CoreAudio требует сборку с CGO")
}
