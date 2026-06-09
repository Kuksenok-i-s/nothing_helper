package bt

import "fmt"

type channelAttemptFunc func(channel int, attempt int) error

// probeRFCOMMChannels tries channelCandidates in order until attempt returns nil.
func probeRFCOMMChannels(preferred int, progress RFCOMMProgress, stepLabel string, attempt channelAttemptFunc) (int, error) {
	var lastErr error
	for i, ch := range channelCandidates(preferred) {
		if i > 0 {
			report(progress, fmt.Sprintf("%s RFCOMM channel %d", stepLabel, ch))
		}
		err := attempt(ch, i)
		if err == nil {
			return ch, nil
		}
		lastErr = err
		if !shouldProbeNextChannel(err) {
			return 0, err
		}
	}
	return 0, fmt.Errorf("%w: %w", ErrRFCOMMNoChannel, lastErr)
}
