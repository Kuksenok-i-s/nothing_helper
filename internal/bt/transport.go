package bt

import "io"

// Transport is the platform-neutral RFCOMM byte stream used by session.
type Transport interface {
	io.ReadWriteCloser
	RemoteMAC() string
	Channel() int
	// String returns a stable log label (e.g. "/dev/rfcomm0" or "rfcomm:AA:BB:…:15").
	String() string
}

type rwcTransport struct {
	io.ReadWriteCloser
	mac     string
	channel int
	label   string
}

func newRWCTransport(rwc io.ReadWriteCloser, mac string, channel int, label string) Transport {
	return &rwcTransport{
		ReadWriteCloser: rwc,
		mac:             mac,
		channel:         channel,
		label:           label,
	}
}

func (t *rwcTransport) RemoteMAC() string { return t.mac }
func (t *rwcTransport) Channel() int      { return t.channel }
func (t *rwcTransport) String() string    { return t.label }

// NewTestTransport wraps rwc for unit tests.
func NewTestTransport(rwc io.ReadWriteCloser, mac string, channel int, label string) Transport {
	return newRWCTransport(rwc, mac, channel, label)
}
