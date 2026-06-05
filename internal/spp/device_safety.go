package spp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	MinScanDelay    = 200 * time.Millisecond
	MaxScanCommands = 32
)

var allowedANCModeValues = map[byte]struct{}{
	0:   {},
	1:   {},
	2:   {},
	3:   {},
	4:   {},
	5:   {},
	7:   {},
	252: {},
	253: {},
	254: {},
	255: {},
}

var ancModeNames = map[string]byte{
	"off":      0, // legacy; BuildANCSetPayload picks 0 vs 5 per model
	"close":    0,
	"strong":   1,
	"medium":   2,
	"weak":     3,
	"smart":    4,
	"adaptive": 4,
	"smart1":   253,
	"smart2":   252,
	// Real Ear devices report/accept 7 for transparency (confirmed via 0xE003 events).
	"transparency": 7,
	"transparent":  7,
	"pass_through": 7,
	"passthrough":  7,
	"comfortable":  255,
}

// MaxEQMode returns the highest valid EQ preset index for a model.
func MaxEQMode(model ModelInfo) byte {
	if ModelSupportsFeature(model, "advance_eq") {
		return 7
	}
	return 3
}

// ValidateANCModeValue checks ANC payload middle byte against the official allowlist.
func ValidateANCModeValue(v byte) error {
	if _, ok := allowedANCModeValues[v]; ok {
		return nil
	}
	return fmt.Errorf("invalid ANC mode value %d; allowed: 0-5, 7, 252, 253, 254, 255", v)
}

// ancOffModeValue returns the SET payload mode byte for disabling ANC.
// Newer firmware reports tab-off as mode 5; Ear (1)/(2) use 0.
func ancOffModeValue(model ModelInfo) byte {
	switch model.Protocol {
	case "EarTwosProtocol", "EarColorProtocol", "GirafarigProtocol", "GligarProtocol",
		"EspeonProtocol", "CorsolaProtocol", "CrobatProtocol", "ElekidProtocol":
		return 5
	}
	switch model.Tier {
	case "B+", "C", "C+":
		return 5
	}
	return 0
}

// ParseANCModeArg parses a named or numeric ANC mode for set_current_noise_reduction payload.
func ParseANCModeArg(s string) (byte, error) {
	key := strings.ToLower(strings.TrimSpace(s))
	if value, ok := ancModeNames[key]; ok {
		return value, nil
	}
	value, err := parseByteValue(s)
	if err != nil {
		return 0, fmt.Errorf("invalid ANC mode %q: %w", s, err)
	}
	if err := ValidateANCModeValue(value); err != nil {
		return 0, err
	}
	return value, nil
}

// ValidateEQModeValue checks EQ preset index for the given model.
func ValidateEQModeValue(model ModelInfo, v byte) error {
	max := MaxEQMode(model)
	if v > max {
		return fmt.Errorf("invalid EQ mode %d for model %s; allowed 0..%d", v, modelLabel(model), max)
	}
	return nil
}

// ParseEQModeArg parses EQ preset index for set_eq_mode.
func ParseEQModeArg(model ModelInfo, s string) (byte, error) {
	value, err := parseByteValue(s)
	if err != nil {
		return 0, fmt.Errorf("invalid EQ mode %q: %w", s, err)
	}
	if err := ValidateEQModeValue(model, value); err != nil {
		return 0, err
	}
	return value, nil
}

func modelLabel(model ModelInfo) string {
	if model.Codename != "" {
		return model.Codename
	}
	return "unknown"
}

// ValidateScanRange ensures a scan only issues safe GET commands with bounded rate.
func ValidateScanRange(start, end uint16, delay time.Duration) error {
	if delay < MinScanDelay {
		return fmt.Errorf("scan delay must be at least %s, got %s", MinScanDelay, delay)
	}
	if end < start {
		return fmt.Errorf("scan end %04x is before start %04x", end, start)
	}
	count := int(end-start) + 1
	if count > MaxScanCommands {
		return fmt.Errorf("scan range has %d commands; maximum is %d", count, MaxScanCommands)
	}
	for cmd := start; cmd <= end; cmd++ {
		if !SafeScanCommand(cmd) {
			return fmt.Errorf("scan command %04x is not a safe GET query (0xC0xx only)", cmd)
		}
	}
	return nil
}

// BuildANCSetPayload builds validated ANC set payload {1, mode, 0}.
func BuildANCSetPayload(model ModelInfo, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("usage: anc set <off|strong|medium|weak|adaptive|transparency|0-5|7|252|253|254|255>")
	}
	key := strings.ToLower(strings.TrimSpace(args[0]))
	if key == "off" || key == "close" {
		return []byte{1, ancOffModeValue(model), 0}, nil
	}
	value, err := ParseANCModeArg(args[0])
	if err != nil {
		return nil, err
	}
	return []byte{1, value, 0}, nil
}

// BuildEQSetPayload builds validated EQ set payload.
func BuildEQSetPayload(model ModelInfo, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("usage: eq set <0-%d>", MaxEQMode(model))
	}
	value, err := ParseEQModeArg(model, args[0])
	if err != nil {
		return nil, err
	}
	return []byte{value}, nil
}

// ParseNumericModeArg is used in tests and strict numeric parsing.
func ParseNumericModeArg(s string) (byte, error) {
	raw := strings.TrimSpace(s)
	parsed, err := strconv.ParseUint(raw, 10, 8)
	if err != nil {
		return 0, err
	}
	return byte(parsed), nil
}
