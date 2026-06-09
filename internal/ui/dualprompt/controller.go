package dualprompt

import (
	"fmt"

	"tws_manager/internal/dualpolicy"
	"tws_manager/internal/session"
	"tws_manager/internal/spp"
)

// Controller tracks dual PC-primary prompt state shared by TUI and Gio.
type Controller struct {
	Mode       dualpolicy.Mode
	HostMAC    string
	HostLoaded bool
	HostErr    string
	Pending    spp.DualDevice
	PendingOK  bool
	Visible    bool
	Interacted bool
}

func (c *Controller) OnDisconnected() {
	c.PendingOK = false
	c.Visible = false
	c.Pending = spp.DualDevice{}
	c.Interacted = false
}

func (c *Controller) EnsureHostMAC() {
	if c.HostMAC != "" || c.HostLoaded {
		return
	}
	c.HostLoaded = true
	mac, err := dualpolicy.HostAdapterMAC()
	if err != nil {
		c.HostErr = err.Error()
		return
	}
	c.HostMAC = mac
}

// OnSnapshot updates prompt state from a session snapshot.
// Returns optional status text for the presenter.
func (c *Controller) OnSnapshot(snap session.Snapshot) (status string) {
	if !snap.Connected {
		c.OnDisconnected()
		return ""
	}
	c.EnsureHostMAC()
	phone, ok := dualpolicy.ShouldPrompt(c.Mode, snap.Model, snap.DualList, c.HostMAC)
	if !ok {
		c.PendingOK = false
		c.Visible = false
		return dualpolicy.HostOwnerStatus(snap.DualList, c.HostMAC)
	}
	c.Pending = phone
	c.PendingOK = true
	if c.Interacted {
		c.Visible = true
	}
	return ""
}

func (c *Controller) OnInteraction() {
	c.Interacted = true
	if c.PendingOK {
		c.Visible = true
	}
}

func (c *Controller) Decline() {
	c.Visible = false
}

func (c *Controller) AcceptFields() ([]string, error) {
	if c.HostMAC == "" {
		if c.HostErr != "" {
			return nil, fmt.Errorf("host bluetooth MAC unavailable: %s", c.HostErr)
		}
		return nil, fmt.Errorf("host bluetooth MAC unavailable")
	}
	c.Visible = false
	return []string{"dual", "connect", c.HostMAC}, nil
}

func (c *Controller) PromptLine() string {
	if !c.Visible || !c.PendingOK {
		return ""
	}
	return dualpolicy.PromptText(c.Pending)
}
