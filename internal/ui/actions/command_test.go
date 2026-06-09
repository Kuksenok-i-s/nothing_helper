package actions

import (
	"testing"

	"tws_manager/internal/ui/presenter"
)

func TestExecuteRejectsScanCommand(t *testing.T) {
	res := Execute(nil, presenter.Command{Title: "Advanced: raw scan", Advanced: true}, ExecOpts{})
	if res.Err == nil {
		t.Fatal("scan command must be rejected by Execute")
	}
}
