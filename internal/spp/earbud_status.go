package spp

import "fmt"

// EarbudStatus is decoded from GET_EARPHONE_STATUS and status notifications.
// Absence from the map means unknown, never "out of ear".
type EarbudStatus struct {
	InEar     bool
	InCase    bool
	Connected bool
}

// Find uses the two-byte side/enabled command on the shared EarTwos protocol.
// Unknown and legacy protocols stay disabled until their payload is verified.
func SupportsFind(model ModelInfo) bool { return model.Protocol == "EarTwosProtocol" }

// Wire reference: https://github.com/Bestello/dms-nothingx/blob/main/main.py
// find_my uses F002 with [2|3, 0|1] for left/right and stop/start.
func FindPayload(side string, enabled bool) ([]byte, error) {
	var id byte
	switch side {
	case "left":
		id = 2
	case "right":
		id = 3
	default:
		return nil, fmt.Errorf("unknown earbud %q", side)
	}
	var on byte
	if enabled {
		on = 1
	}
	return []byte{id, on}, nil
}

func ValidateFindPayload(payload []byte) error {
	if len(payload) != 2 || (payload[0] != 2 && payload[0] != 3) || payload[1] > 1 {
		return fmt.Errorf("invalid find payload: expected left/right and on/off")
	}
	return nil
}
