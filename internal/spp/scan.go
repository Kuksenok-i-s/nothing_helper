package spp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseHexCommand(raw string) (uint16, error) {
	raw = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "0x")
	cmd, err := strconv.ParseUint(raw, 16, 16)
	if err != nil {
		return 0, err
	}

	return uint16(cmd), nil
}

func ParseScanCommand(fields []string) (uint16, uint16, time.Duration, error) {
	if len(fields) != 4 {
		return 0, 0, 0, fmt.Errorf("usage: scan <start_hex> <end_hex> <delay>")
	}

	start, err := ParseHexCommand(fields[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid scan start %q: %w", fields[1], err)
	}

	end, err := ParseHexCommand(fields[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid scan end %q: %w", fields[2], err)
	}

	delay, err := time.ParseDuration(fields[3])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid scan delay %q: %w", fields[3], err)
	}

	if end < start {
		return 0, 0, 0, fmt.Errorf("scan end %04x is before start %04x", end, start)
	}

	return start, end, delay, nil
}

func SafeScanCommand(cmd uint16) bool {
	return cmd&0xF000 == 0xC000
}

