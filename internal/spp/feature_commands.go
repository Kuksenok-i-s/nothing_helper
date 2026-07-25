package spp

import (
	"fmt"
	"strconv"
	"strings"
)

type FeatureCommand struct {
	Name            string
	Feature         string
	GetCommand      uint16
	GetPayload      []byte
	SetCommand      uint16
	SafeSet         bool
	Usage           string
	BuildSetPayload func(model ModelInfo, args []string) ([]byte, error)
}

var featureCommands = map[string]FeatureCommand{
	"anc": {
		Name:            "ANC",
		Feature:         "anc",
		GetCommand:      CmdGetNoiseReduction,
		GetPayload:      []byte{3},
		SetCommand:      CmdSetNoiseReduction,
		SafeSet:         true,
		Usage:           "anc [get] | anc set <off|strong|medium|weak|adaptive|transparency>",
		BuildSetPayload: BuildANCSetPayload,
	},
	"eq": {
		Name:            "EQ mode",
		Feature:         "eq",
		GetCommand:      CmdGetEQMode,
		SetCommand:      CmdSetEQMode,
		SafeSet:         true,
		Usage:           "eq [get] | eq set <0-N preset>",
		BuildSetPayload: BuildEQSetPayload,
	},
	"spatial": {
		Name:       "Spatial audio",
		Feature:    "spatial",
		GetCommand: CmdGetSpatialAudio,
		SetCommand: CmdSetSpatialAudio,
		SafeSet:    true,
		Usage:      "spatial [get] | spatial set <on|off|1|0> [head-track-on|off]",
		BuildSetPayload: func(model ModelInfo, args []string) ([]byte, error) {
			if len(args) != 1 && len(args) != 2 {
				return nil, fmt.Errorf("usage: spatial set <on|off|1|0> [head-track-on|off]")
			}

			enabled, err := parseBoolByte(args[0])
			if err != nil {
				return nil, err
			}
			if len(args) == 1 {
				return []byte{enabled}, nil
			}

			headTrack, err := parseBoolByte(args[1])
			if err != nil {
				return nil, err
			}

			return []byte{enabled, headTrack}, nil
		},
	},
	"lag": {
		Name:       "Low latency",
		Feature:    "lag",
		GetCommand: CmdGetLagMode,
		SetCommand: CmdSetLagMode,
		SafeSet:    true,
		Usage:      "lag [get] | lag set <on|off|1|0>",
		BuildSetPayload: func(model ModelInfo, args []string) ([]byte, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("usage: lag set <on|off|1|0>")
			}
			enabled, err := parseBoolByte(args[0])
			if err != nil {
				return nil, err
			}
			if enabled == 1 {
				return []byte{1}, nil
			}
			return []byte{2}, nil
		},
	},
	"dual": {
		Name:       "Dual connection",
		Feature:    "dual",
		GetCommand: CmdGetDualEnable,
		SetCommand: CmdSetDualEnable,
		SafeSet:    true,
		Usage:      "dual [get] | dual list | dual set <on|off|1|0> | dual connect <mac> | dual disconnect <mac>",
		BuildSetPayload: func(model ModelInfo, args []string) ([]byte, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("usage: dual set <on|off|1|0>")
			}
			enabled, err := parseBoolByte(args[0])
			if err != nil {
				return nil, err
			}
			return []byte{enabled}, nil
		},
	},
}

func parseByteValue(value string) (byte, error) {
	raw := strings.TrimSpace(value)
	base := 10
	if strings.HasPrefix(strings.ToLower(raw), "0x") {
		base = 16
		raw = raw[2:]
	}

	parsed, err := strconv.ParseUint(raw, base, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid byte value %q", value)
	}

	return byte(parsed), nil
}

func parseBoolByte(value string) (byte, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "on", "true", "yes", "enable", "enabled":
		return 1, nil
	case "0", "off", "false", "no", "disable", "disabled":
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid boolean value %q", value)
	}
}

func parseFeatureFields(fields []string) (FeatureCommand, string, []string, error) {
	spec, ok := featureCommands[strings.ToLower(fields[0])]
	if !ok {
		return FeatureCommand{}, "", nil, fmt.Errorf("unknown feature command %q", fields[0])
	}
	action := "get"
	args := fields[1:]
	if len(args) > 0 {
		action = strings.ToLower(args[0])
		args = args[1:]
	}
	return spec, action, args, nil
}

func featureWarnings(model ModelInfo, spec FeatureCommand) []string {
	if ModelSupportsFeature(model, spec.Feature) {
		return nil
	}
	return []string{fmt.Sprintf("%s is not listed for model %s tier %s", spec.Name, model.Codename, model.Tier)}
}

func buildFeatureGetPacket(spec FeatureCommand, args []string) (Packet, error) {
	if len(args) != 0 {
		return Packet{}, fmt.Errorf("usage: %s", spec.Usage)
	}
	return Packet{
		Cmd:     spec.GetCommand,
		Payload: append([]byte(nil), spec.GetPayload...),
	}, nil
}

func buildFeatureListPacket(spec FeatureCommand, args []string) (Packet, error) {
	if spec.Feature != "dual" || len(args) != 0 {
		return Packet{}, fmt.Errorf("usage: %s", spec.Usage)
	}
	return Packet{Cmd: CmdGetDualDeviceList, Payload: []byte{0}}, nil
}

func buildFeatureDualConnectPacket(spec FeatureCommand, action string, args []string, allowUnsafe bool) (Packet, []string, error) {
	if err := validateDualConnectRequest(spec, action, args, allowUnsafe); err != nil {
		return Packet{}, nil, err
	}
	payload, warnings, err := BuildDualConnectPayload(action == "connect", args[0])
	if err != nil {
		return Packet{}, nil, err
	}
	return Packet{Cmd: CmdSetConnectDevice, Payload: payload}, warnings, nil
}

func validateDualConnectRequest(spec FeatureCommand, action string, args []string, allowUnsafe bool) error {
	if !allowUnsafe && !spec.SafeSet {
		return fmt.Errorf("dual %s writes to the device; re-run with --unsafe to allow it", action)
	}
	if spec.Feature != "dual" {
		return fmt.Errorf("usage: %s", spec.Usage)
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: dual %s <mac>", action)
	}
	return nil
}

func buildFeatureSetPacket(spec FeatureCommand, args []string, allowUnsafe bool, model ModelInfo) (Packet, error) {
	if !allowUnsafe && !spec.SafeSet {
		return Packet{}, fmt.Errorf("%s set writes to the device; re-run with --unsafe to allow it", spec.Name)
	}
	if model.Codename == "" {
		return Packet{}, fmt.Errorf("%s set requires a known model; pass --model or connect through discovery so it can be auto-detected", spec.Name)
	}
	payload, err := spec.BuildSetPayload(model, args)
	if err != nil {
		return Packet{}, err
	}
	return Packet{Cmd: spec.SetCommand, Payload: payload}, nil
}

func FeatureCommandPacket(fields []string, allowUnsafe bool, model ModelInfo) (Packet, []string, error) {
	if len(fields) == 0 {
		return Packet{}, nil, fmt.Errorf("missing feature command")
	}

	spec, action, args, err := parseFeatureFields(fields)
	if err != nil {
		return Packet{}, nil, err
	}

	warnings := featureWarnings(model, spec)
	return buildFeatureActionPacket(spec, action, args, allowUnsafe, model, warnings)
}

func buildFeatureActionPacket(spec FeatureCommand, action string, args []string, allowUnsafe bool, model ModelInfo, warnings []string) (Packet, []string, error) {
	switch action {
	case "get":
		pkt, err := buildFeatureGetPacket(spec, args)
		return pkt, warnings, err
	case "list":
		pkt, err := buildFeatureListPacket(spec, args)
		return pkt, warnings, err
	case "connect", "disconnect":
		pkt, connectWarnings, err := buildFeatureDualConnectPacket(spec, action, args, allowUnsafe)
		warnings = append(warnings, connectWarnings...)
		return pkt, warnings, err
	case "set":
		pkt, err := buildFeatureSetPacket(spec, args, allowUnsafe, model)
		return pkt, warnings, err
	default:
		return Packet{}, warnings, fmt.Errorf("usage: %s", spec.Usage)
	}
}

func FeatureCommands() map[string]FeatureCommand {
	out := make(map[string]FeatureCommand, len(featureCommands))
	for k, v := range featureCommands {
		out[k] = v
	}
	return out
}
