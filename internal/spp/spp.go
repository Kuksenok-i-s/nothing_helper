package spp

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	CmdGetProtocolVersion    = 0xC001
	CmdGetStatus             = 0xC00A
	CmdGetFirmwareVersion    = 0xC042
	CmdGetBattery            = 0xC007
	CmdGetIdentity           = 0xC005
	CmdGetRemoteConfig       = 0xC006
	CmdGetSupportedFeature   = 0xC00D
	CmdGetEQMode             = 0xC01F
	CmdGetNoiseReduction     = 0xC01E
	CmdGetSpatialAudio       = 0xC04F
	CmdGetLagMode            = 0xC041
	CmdGetDualEnable         = 0xC027
	CmdGetDualDeviceList     = 0xC028
	CmdSetEQMode             = 0xF010
	CmdSetNoiseReduction     = 0xF00F
	CmdSetSpatialAudio       = 0xF052
	CmdSetLagMode            = 0xF040
	CmdSetDualEnable         = 0xF01A
	CmdSetConnectDevice      = 0xF01B
	CmdAckSetLagMode         = 0x7040
	CmdAckSetNoiseReduction  = 0x700F
	CmdAckSetEQMode          = 0x7010
	CmdAckSetDualEnable      = 0x701A
	CmdAckSetConnectDevice   = 0x701B
	CmdAckSetSpatialAudio    = 0x7052
	CmdBatteryChanged        = 0xE001
	CmdNoiseReductionChanged = 0xE003
	CmdDualSwitchChanged     = 0xE006
	CmdBattery               = 0xE007
	CmdBudsBattery           = 0xE005
	CmdStatus                = 0xE002
	CmdIdentity              = 0xE008
	CmdDualConnectChanged    = 0xE00E
	CmdLagModeChanged        = 0xE019
)

var partNames = map[byte]string{
	1: "watch",
	2: "left",
	3: "right",
	4: "case",
	5: "tws",
	6: "stereo",
}

type Battery struct {
	Percent  int  `json:"percent"`
	Charging bool `json:"charging,omitempty"`
}

type CommandInfo struct {
	Name   string
	Kind   string
	Safe   bool
	Unsafe bool
}

var commandCatalog = map[uint16]CommandInfo{
	0x4001: {Name: "rsp_protocol_version", Kind: "text"},
	0x4005: {Name: "rsp_device_identification", Kind: "identity"},
	0x4006: {Name: "rsp_remote_configuration", Kind: "binary"},
	0x4007: {Name: "rsp_battery_level", Kind: "battery_pairs"},
	0x400A: {Name: "rsp_earphone_status", Kind: "binary_status"},
	0x400D: {Name: "rsp_supported_feature", Kind: "binary"},
	0x401E: {Name: "rsp_noise_reduction", Kind: "binary"},
	0x401F: {Name: "rsp_eq_mode", Kind: "binary"},
	0x4027: {Name: "rsp_dual_enable", Kind: "binary"},
	0x4028: {Name: "rsp_dual_device_list", Kind: "binary"},
	0x4041: {Name: "rsp_host_lag_mode", Kind: "binary"},
	0x4042: {Name: "rsp_host_version", Kind: "text"},
	0x404F: {Name: "rsp_spatial_audio", Kind: "binary"},
	0x700F: {Name: "ack_set_noise_reduction", Kind: "ack"},
	0x7010: {Name: "ack_set_eq_mode", Kind: "ack"},
	0x701A: {Name: "ack_set_dual_enable", Kind: "ack"},
	0x701B: {Name: "ack_set_connect_device", Kind: "ack"},
	0x7040: {Name: "ack_set_lag_mode", Kind: "ack"},
	0x7052: {Name: "ack_set_spatial_audio", Kind: "ack"},

	0xC001: {Name: "get_protocol_version", Kind: "query"},
	0xC002: {Name: "get_find_ear_state", Kind: "query"},
	0xC003: {Name: "get_remote_mtu", Kind: "query"},
	0xC004: {Name: "get_remote_vid", Kind: "query"},
	0xC005: {Name: "get_remote_device_identification", Kind: "query"},
	0xC006: {Name: "get_remote_configuration", Kind: "query"},
	0xC007: {Name: "get_remote_battery_level", Kind: "query"},
	0xC008: {Name: "get_upgrade_capability", Kind: "query"},
	0xC009: {Name: "get_supported_gesture", Kind: "query"},
	0xC00A: {Name: "get_earphone_status", Kind: "query"},
	0xC00B: {Name: "get_remote_extra_version_code", Kind: "query"},
	0xC00C: {Name: "get_remote_color_id", Kind: "query"},
	0xC00D: {Name: "get_supported_feature", Kind: "query"},
	0xC00E: {Name: "get_extra_feature_status", Kind: "query"},
	0xC00F: {Name: "get_eq_id", Kind: "query"},
	0xC010: {Name: "get_high_volume_gain_level", Kind: "query"},
	0xC011: {Name: "get_auto_power_off_time", Kind: "query"},
	0xC013: {Name: "get_earphone_connected_status", Kind: "query"},
	0xC014: {Name: "get_volume", Kind: "query"},
	0xC015: {Name: "get_codec_capability", Kind: "query"},
	0xC016: {Name: "get_manufacture", Kind: "query"},
	0xC017: {Name: "get_box_led_color", Kind: "query"},
	0xC018: {Name: "get_key_configuration", Kind: "query"},
	0xC019: {Name: "get_device_working_status", Kind: "query"},
	0xC01C: {Name: "get_device_model", Kind: "query"},
	0xC01D: {Name: "get_noise_reduction_configuration", Kind: "query"},
	0xC01E: {Name: "get_current_noise_reduction", Kind: "query"},
	0xC01F: {Name: "get_eq_mode", Kind: "query"},
	0xC020: {Name: "get_personalized_anc", Kind: "query"},
	0xC021: {Name: "get_personalized_noise_value", Kind: "query"},
	0xC022: {Name: "get_mimi_enable", Kind: "query"},
	0xC023: {Name: "get_mimi_intensity", Kind: "query"},
	0xC024: {Name: "get_mimi_preset_id", Kind: "query"},
	0xC025: {Name: "get_mimi_fitting_tech_level", Kind: "query"},
	0xC026: {Name: "get_3d_mode", Kind: "query"},
	0xC027: {Name: "get_dual_enable", Kind: "query"},
	0xC028: {Name: "get_dual_device_list", Kind: "query"},
	0xC029: {Name: "get_lhdc_commands", Kind: "query"},
	0xC03D: {Name: "get_supported_notification", Kind: "query"},
	0xC03E: {Name: "get_registered_notification", Kind: "query"},
	0xC03F: {Name: "get_host_utc_time", Kind: "query"},
	0xC041: {Name: "get_host_lag_mode", Kind: "query"},
	0xC042: {Name: "get_host_version_device", Kind: "query"},
	0xC043: {Name: "get_adaptive_eq_mode", Kind: "query"},
	0xC044: {Name: "get_custom_eq_value", Kind: "query"},
	0xC04C: {Name: "get_advance_custom_eq_mode", Kind: "query"},
	0xC04D: {Name: "get_advance_custom_eq_value", Kind: "query"},
	0xC04E: {Name: "get_bass_boost", Kind: "query"},
	0xC04F: {Name: "get_spatial_audio", Kind: "query"},
	0xC050: {Name: "get_dirac_opteo_eq", Kind: "query"},
	0xC051: {Name: "get_anc_fir_mode", Kind: "query"},
	0xC053: {Name: "get_bass_enhancer_mode", Kind: "query"},
	0xC054: {Name: "get_smart_free_mode", Kind: "query"},
	0xC055: {Name: "get_smart_anc_mode", Kind: "query"},
	0xC056: {Name: "get_le_switch", Kind: "query"},
	0xC057: {Name: "get_system_audio", Kind: "query"},
	0xC058: {Name: "get_headtrack_start", Kind: "query"},
	0xC059: {Name: "get_le_audio_connect_mode", Kind: "query"},
	0xC05A: {Name: "get_box_version", Kind: "query"},
	0xC062: {Name: "get_mutually_exclusive", Kind: "query"},
	0xC063: {Name: "get_sky_walk_support", Kind: "query"},

	0xE001: {Name: "event_battery_changed", Kind: "battery_pairs"},
	0xE002: {Name: "event_device_status_changed", Kind: "binary_status"},
	0xE003: {Name: "event_noise_reduction_level_changed", Kind: "notification"},
	0xE005: {Name: "event_game_mode_changed", Kind: "notification"},
	0xE006: {Name: "event_dual_device_switch_state", Kind: "notification"},
	0xE007: {Name: "battery_full", Kind: "battery_pairs"},
	0xE008: {Name: "identity", Kind: "identity"},
	0xE009: {Name: "event_working_status_change", Kind: "notification"},
	0xE00B: {Name: "event_led_color_sync_notification", Kind: "notification"},
	0xE00C: {Name: "event_personalize_sync_notification", Kind: "notification"},
	0xE00D: {Name: "event_tip_fit_result", Kind: "notification"},
	0xE00E: {Name: "event_dual_device_connect_state", Kind: "notification"},
	0xE00F: {Name: "notify_disconnect_profile", Kind: "notification"},
	0xE010: {Name: "notify_request_start_ota", Kind: "notification"},
	0xE011: {Name: "notify_request_stop_ota", Kind: "notification"},
	0xE014: {Name: "event_magic_button", Kind: "notification"},
	0xE015: {Name: "event_head_track", Kind: "notification"},
	0xE016: {Name: "event_le_audio_connect", Kind: "notification"},
	0xE018: {Name: "event_recording", Kind: "notification"},
	0xE019: {Name: "event_lag_mode_changed", Kind: "notification"},

	0xF001: {Name: "set_protocol_activated", Kind: "set"},
	0xF002: {Name: "set_where_am_i", Kind: "set"},
	0xF003: {Name: "set_key_configuration", Kind: "set"},
	0xF004: {Name: "set_extra_feature_status", Kind: "set"},
	0xF007: {Name: "set_eq_status", Kind: "set"},
	0xF008: {Name: "set_high_volume_gain_level", Kind: "set"},
	0xF00A: {Name: "set_utc_time", Kind: "set"},
	0xF00B: {Name: "set_auto_power_off_time", Kind: "set"},
	0xF00D: {Name: "set_box_led_color", Kind: "set"},
	0xF00E: {Name: "set_noise_reduction_configuration", Kind: "set"},
	0xF00F: {Name: "set_current_noise_reduction", Kind: "set", Safe: true},
	0xF010: {Name: "set_eq_mode", Kind: "set", Safe: true},
	0xF011: {Name: "set_personalized", Kind: "set"},
	0xF012: {Name: "set_calibration", Kind: "set"},
	0xF013: {Name: "set_calibration_force", Kind: "set"},
	0xF014: {Name: "set_leak_detect", Kind: "set"},
	0xF015: {Name: "set_mimi_enable", Kind: "set"},
	0xF016: {Name: "set_mimi_intensity", Kind: "set"},
	0xF017: {Name: "set_mimi_preset_payload", Kind: "set"},
	0xF018: {Name: "set_mimi_preset_id", Kind: "set"},
	0xF019: {Name: "set_3d_sound", Kind: "set"},
	0xF01A: {Name: "set_dual_enable", Kind: "set", Safe: true},
	0xF01B: {Name: "set_connect_device", Kind: "set", Safe: true},
	0xF01C: {Name: "set_lhdc_commands", Kind: "set"},
	0xF01D: {Name: "set_dirac_opteo_eq", Kind: "set"},
	0xF03D: {Name: "restore_factory_setting", Kind: "set", Unsafe: true},
	0xF03E: {Name: "register_notification", Kind: "set"},
	0xF03F: {Name: "unregister_notification", Kind: "set"},
	0xF040: {Name: "set_lag_mode", Kind: "set", Safe: true},
	0xF041: {Name: "set_custom_eq", Kind: "set"},
	0xF042: {Name: "set_adaptive_eq", Kind: "set"},
	0xF044: {Name: "ota_find_new_version", Kind: "set"},
	0xF045: {Name: "ota_downloaded_new_version", Kind: "set"},
	0xF046: {Name: "ota_stop_error", Kind: "set"},
	0xF04F: {Name: "set_advance_custom_eq_mode", Kind: "set"},
	0xF050: {Name: "set_advance_custom_eq_value", Kind: "set"},
	0xF051: {Name: "set_bass_boost", Kind: "set"},
	0xF052: {Name: "set_spatial_audio", Kind: "set", Safe: true},
	0xF053: {Name: "set_fir_anc_mode", Kind: "set"},
	0xF057: {Name: "set_bass_enhancer_mode", Kind: "set"},
	0xF058: {Name: "set_smart_free_mode", Kind: "set"},
	0xF059: {Name: "set_smart_anc_mode", Kind: "set"},
	0xF05A: {Name: "set_le_switch_model", Kind: "set"},
	0xF05B: {Name: "set_system_audio", Kind: "set"},
	0xF062: {Name: "set_essential_space_status", Kind: "set"},

	0xFC01: {Name: "debug_enter_test_mode", Kind: "debug", Unsafe: true},
	0xFC02: {Name: "debug_parameter_negotiation", Kind: "debug", Unsafe: true},
	0xFC03: {Name: "debug_get_file_list", Kind: "debug"},
	0xFC04: {Name: "debug_query_single_file_info", Kind: "debug"},
	0xFC05: {Name: "debug_request_single_file_info", Kind: "debug"},
	0xFC06: {Name: "debug_device_send_data", Kind: "debug", Unsafe: true},
	0xFC07: {Name: "debug_exit_test_mode", Kind: "debug", Unsafe: true},
	0xFC08: {Name: "debug_change_level", Kind: "debug", Unsafe: true},
	0xFC09: {Name: "debug_get_debug_info", Kind: "debug"},
	0xFC20: {Name: "buried_device_send", Kind: "debug"},
	0xFC21: {Name: "buried_log_info", Kind: "debug"},
	0xFC22: {Name: "buried_log_request", Kind: "debug"},
	0xFC30: {Name: "debug_curve_command", Kind: "debug"},
}

func CommandInfoFor(cmd uint16) CommandInfo {
	if info, ok := commandCatalog[cmd]; ok {
		return info
	}

	return CommandInfo{Name: "unknown", Kind: "unknown"}
}

func CommandLabel(cmd uint16) string {
	info := CommandInfoFor(cmd)
	if info.Name == "unknown" {
		return fmt.Sprintf("%04x", cmd)
	}
	return fmt.Sprintf("%04x/%s", cmd, info.Name)
}

func CommandCatalog() map[uint16]CommandInfo {
	out := make(map[uint16]CommandInfo, len(commandCatalog))
	for k, v := range commandCatalog {
		out[k] = v
	}
	return out
}

type ModelInfo struct {
	Codename          string
	Product           string
	FastPairID        string
	Protocol          string
	Tier              string
	BatteryCaseSource string
	Features          []string
	Aliases           []string
}

var knownModels = []ModelInfo{
	{Codename: "EarOne", Product: "Nothing ear (1)", FastPairID: "31D53D", Protocol: "EarOneProtocol", Tier: "A", BatteryCaseSource: "case", Features: []string{"anc", "eq"}, Aliases: []string{"ear one", "ear1", "ear (1)", "nothing ear (1)"}},
	{Codename: "EarTwo", Product: "Ear (2)", FastPairID: "DEE8C0", Protocol: "EarTwoProtocol", Tier: "B", BatteryCaseSource: "case", Features: []string{"anc", "eq", "dual", "lhdc", "mimi", "3d"}, Aliases: []string{"ear two", "ear2", "ear (2)"}},
	{Codename: "EarTwos", Product: "Nothing Ear (2024)", FastPairID: "FEB1C7", Protocol: "EarTwosProtocol", Tier: "B+", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "dual", "advance_eq", "mimi", "3d"}, Aliases: []string{"ear twos", "nothing ear", "ear 2024"}},
	{Codename: "EarThree", Product: "Ear (3)", FastPairID: "C1EBFD", Protocol: "EarTwosProtocol", Tier: "B+", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "dual", "advance_eq", "bass"}, Aliases: []string{"ear three", "ear3", "ear (3)", "nothing ear (3)", "feraligatr"}},
	{Codename: "EarStick", Product: "Ear (stick)", FastPairID: "1016DD", Protocol: "EarStickProtocol", Tier: "B-", BatteryCaseSource: "case", Features: []string{"eq", "advance_eq"}, Aliases: []string{"ear stick", "ear (stick)"}},
	{Codename: "EarColor", Product: "Nothing Ear (a)", FastPairID: "5E3FBC", Protocol: "EarColorProtocol", Tier: "B", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "dual", "advance_eq", "3d"}, Aliases: []string{"ear color", "ear a", "ear (a)", "nothing ear (a)"}},
	{Codename: "Flaffy", Product: "Nothing ear (open)", FastPairID: "FC3AAF", Protocol: "FlaffyProtocol", Tier: "B", BatteryCaseSource: "case", Features: []string{"eq", "dual", "advance_eq", "3d"}, Aliases: []string{"ear open", "ear (open)", "nothing ear (open)", "cc3444"}},
	{Codename: "Elekid", Product: "Nothing Headphone (1)", FastPairID: "2D6FDA", Protocol: "ElekidProtocol", Tier: "C", BatteryCaseSource: "stereo", Features: []string{"anc", "eq", "spatial", "dual", "bass"}, Aliases: []string{"headphone 1", "headphone (1)", "nothing headphone (1)"}},
	{Codename: "Forretress", Product: "Headphone Pro", FastPairID: "73C9EB", Protocol: "ElekidProtocol", Tier: "C+", BatteryCaseSource: "stereo", Features: []string{"anc", "eq", "spatial", "headtrack", "le_audio", "system_audio", "magic_button", "bass"}, Aliases: []string{"forretress", "headphone pro", "24211"}},
	{Codename: "Crobat", Product: "CMF Neckband Pro", FastPairID: "AE35FD", Protocol: "CrobatProtocol", Tier: "C", BatteryCaseSource: "stereo", Features: []string{"anc", "eq", "spatial"}, Aliases: []string{"neckband pro", "cmf neckband pro"}},
	{Codename: "Corsola", Product: "CMF Buds Pro", FastPairID: "ADD2C4", Protocol: "CorsolaProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial"}, Aliases: []string{"buds pro", "cmf buds pro"}},
	{Codename: "Donphan", Product: "CMF Buds", FastPairID: "150A27", Protocol: "DonphanProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"eq"}, Aliases: []string{"cmf buds", "buds"}},
	{Codename: "Espeon", Product: "CMF Buds Pro 2", FastPairID: "F29566", Protocol: "EspeonProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "bass"}, Aliases: []string{"buds pro 2", "cmf buds pro 2"}},
	{Codename: "Girafarig", Product: "24232", FastPairID: "19EF24", Protocol: "GirafarigProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "bass"}, Aliases: []string{"24232"}},
	{Codename: "Gligar", Product: "24241", FastPairID: "4AEB6E", Protocol: "GligarProtocol", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq", "spatial", "bass"}, Aliases: []string{"24241"}},
	{Codename: "Hitmontop", Product: "24272", FastPairID: "404D6D", Protocol: "unpublished", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq"}, Aliases: []string{"24272"}},
	{Codename: "Hoothoot", Product: "24283", FastPairID: "70F8E3", Protocol: "unpublished", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq"}, Aliases: []string{"24283"}},
	{Codename: "Heracross", Product: "24253", FastPairID: "2F45F5", Protocol: "unpublished", Tier: "C", BatteryCaseSource: "case", Features: []string{"anc", "eq"}, Aliases: []string{"24253"}},
}

var activeModel ModelInfo

func normalizeModelKey(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", " "))
}

func ResolveModelInfo(value string) (ModelInfo, bool) {
	key := normalizeModelKey(value)
	if key == "" {
		return ModelInfo{}, false
	}

	for _, model := range knownModels {
		if normalizeModelKey(model.Codename) == key ||
			normalizeModelKey(model.Product) == key ||
			normalizeModelKey(model.FastPairID) == key {
			return model, true
		}

		for _, alias := range model.Aliases {
			if normalizeModelKey(alias) == key {
				return model, true
			}
		}
	}

	return ModelInfo{}, false
}

func ResolveModelFromBluetooth(values ...string) (ModelInfo, string, bool) {
	for _, value := range values {
		if model, ok := ResolveModelInfo(value); ok {
			return model, strings.TrimSpace(value), true
		}
	}

	haystack := strings.ToUpper(strings.Join(values, "\n"))
	for _, model := range knownModels {
		fastPairID := strings.ToUpper(strings.TrimSpace(model.FastPairID))
		if fastPairID != "" && strings.Contains(haystack, fastPairID) {
			return model, "fast_pair_id:" + fastPairID, true
		}
	}

	normalizedHaystack := normalizeModelKey(strings.Join(values, "\n"))
	for _, model := range knownModels {
		for _, candidate := range append([]string{model.Product, model.Codename}, model.Aliases...) {
			key := normalizeModelKey(candidate)
			if len(key) >= 6 && strings.Contains(normalizedHaystack, key) {
				return model, strings.TrimSpace(candidate), true
			}
		}
	}

	return ModelInfo{}, "", false
}

func ModelSupportsFeature(model ModelInfo, feature string) bool {
	if model.Codename == "" {
		return true
	}
	if feature == "lag" {
		return true
	}

	for _, item := range model.Features {
		if item == feature {
			return true
		}
	}

	return false
}

func KnownModels() []ModelInfo {
	out := make([]ModelInfo, len(knownModels))
	copy(out, knownModels)
	return out
}

func DefaultModel() ModelInfo { return ModelInfo{BatteryCaseSource: "case"} }

func readExact(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func DecodePacket(raw []byte) (Packet, error) {
	if len(raw) < 8 {
		return Packet{Raw: raw}, fmt.Errorf("packet too short: %d bytes", len(raw))
	}

	if raw[0] != SOF {
		return Packet{Raw: raw}, fmt.Errorf("invalid SOF: %02x", raw[0])
	}

	control := getUint16LE(raw, 1)
	cmd := getUint16LE(raw, 3)
	length := getUint16LE(raw, 5)
	fsn := raw[7]

	want := 8 + int(length)
	if control&ControlCRC != 0 {
		want += 2
	}
	if len(raw) < want {
		return Packet{Raw: raw}, fmt.Errorf("truncated payload: length=%d raw_len=%d", length, len(raw))
	}

	pkt := Packet{
		Control: control,
		Cmd:     cmd,
		Length:  length,
		FSN:     fsn,
		Raw:     append([]byte(nil), raw...),
	}

	if length > 0 {
		pkt.Payload = append([]byte(nil), raw[8:8+length]...)
	}

	if control&ControlCRC != 0 {
		off := 8 + int(length)
		pkt.CRC = getUint16LE(raw, off)
		computed := CRC16(raw[:off])
		pkt.CRCValid = computed == pkt.CRC
	}

	return pkt, nil
}

func ParsePairs(payload []byte) map[string]Battery {
	if len(payload) == 0 || len(payload)%2 != 0 {
		return nil
	}

	result := map[string]Battery{}

	for i := 0; i < len(payload); i += 2 {
		kind := payload[i]
		val := payload[i+1]

		name, ok := partNames[kind]
		if !ok {
			return nil
		}

		result[name] = Battery{
			Percent:  int(val & 0x7F),
			Charging: val&0x80 != 0,
		}
	}

	return result
}

func FormatBattery(data map[string]Battery) string {
	required := []string{"left", "right", "case"}
	extras := []string{"watch", "tws", "stereo"}
	out := ""

	appendPart := func(part string) {
		if out == "" {
			out = part
		} else {
			out += " | " + part
		}
	}

	formatPart := func(name string, item Battery) string {
		part := fmt.Sprintf("%s: %d%%", name, item.Percent)
		if item.Charging {
			part += " charging"
		}

		return part
	}

	for _, name := range required {
		item, ok := data[name]
		if !ok {
			appendPart(fmt.Sprintf("%s: n/a", name))
			continue
		}

		appendPart(formatPart(name, item))
	}

	for _, name := range extras {
		item, ok := data[name]
		if !ok {
			continue
		}

		appendPart(formatPart(name, item))
	}

	return out
}

func NormalizeBatteryForModel(data map[string]Battery, model ModelInfo) (map[string]Battery, []string) {
	if data == nil {
		return nil, nil
	}

	out := make(map[string]Battery, len(data)+1)
	for key, value := range data {
		out[key] = value
	}

	var warnings []string
	if model.BatteryCaseSource != "stereo" {
		return out, warnings
	}

	stereo, hasStereo := out["stereo"]
	if !hasStereo {
		return out, warnings
	}

	if _, hasCase := out["case"]; hasCase {
		warnings = append(warnings, "model uses stereo battery as case battery, but payload also includes case; keeping explicit case value")
		return out, warnings
	}

	out["case"] = stereo
	delete(out, "stereo")

	if model.Codename != "" {
		warnings = append(warnings, fmt.Sprintf("%s maps stereo battery (id=6) to case battery", model.Codename))
	}

	return out, warnings
}

type ParsedPacket struct {
	Kind      string
	Summary   string
	Text      string
	Batteries map[string]Battery
	DualList  *DualDeviceList
	Warnings  []string
}

// DualDevice is one peer entry from GET_DUAL_DEVICE_LIST (0x4028).
type DualDevice struct {
	MAC       string
	Name      string
	Connected bool
	Owner     bool
	RawState  byte
}

// DualDeviceList is the parsed payload of GET_DUAL_DEVICE_LIST.
type DualDeviceList struct {
	Total      byte
	Current    byte
	Devices    []DualDevice
	RawPayload []byte
}

type PacketParser func(ModelInfo) func(Packet) ParsedPacket

var packetParsers = map[uint16]PacketParser{
	CmdBatteryChanged:        parseBatteryPacket("battery_changed"),
	CmdBattery:               parseBatteryPacket("battery_full"),
	CmdBudsBattery:           parseBatteryPacket("battery_buds"),
	CmdStatus:                parseStatusPacket("status_changed"),
	CmdNoiseReductionChanged: parseANCPacket("anc_changed"),
	CmdDualSwitchChanged:     parseDualPacket("dual_switch_changed"),
	CmdDualConnectChanged:    parseDualPacket("dual_connect_changed"),
	CmdLagModeChanged:        parseLagPacket("lag_changed"),
	CmdIdentity:              parseRawPacket("identity_raw"),

	CmdRspBattery:          parseBatteryPacket("battery_response"),
	CmdRspStatus:           parseStatusPacket("status_response"),
	CmdRspIdentity:         parseRawPacket("identity_response"),
	CmdRspRemoteConfig:     parseConfigPacket("config_response"),
	CmdRspSupportedFeature: parseSupportedFeaturePacket("supported_features"),
	CmdRspFirmware:         parseTextPacket("firmware/version"),
	CmdRspProtocolVersion:  parseTextPacket("protocol/version"),
	CmdRspANC:              parseANCPacket("anc_response"),
	CmdRspEQ:               parseEQPacket("eq_response"),
	CmdRspDualEnable:       parseDualPacket("dual_response"),
	CmdRspDualDeviceList:   parseDualDeviceListPacket("dual_device_list"),
	CmdRspLag:              parseLagPacket("lag_response"),
	CmdRspSpatial:          parseSpatialPacket("spatial_response"),

	CmdAckSetLagMode:        parseSetAckPacket("lag_set_ack"),
	CmdAckSetNoiseReduction: parseSetAckPacket("anc_set_ack"),
	CmdAckSetEQMode:         parseSetAckPacket("eq_set_ack"),
	CmdAckSetSpatialAudio:   parseSetAckPacket("spatial_set_ack"),
	CmdAckSetDualEnable:     parseSetAckPacket("dual_enable_set_ack"),
	CmdAckSetConnectDevice:  parseSetAckPacket("dual_connect_set_ack"),
}

// parseCountedPairs decodes the counted "[count][id,val]*count" layout used by
// battery/status responses (DataExtKt.toPairs with count=1, id=1, val=1).
func parseCountedPairs(payload []byte) ([][2]byte, bool) {
	if len(payload) < 1 {
		return nil, false
	}
	count := int(payload[0])
	if len(payload) < 1+count*2 {
		return nil, false
	}
	out := make([][2]byte, 0, count)
	for i := 0; i < count; i++ {
		off := 1 + i*2
		out = append(out, [2]byte{payload[off], payload[off+1]})
	}
	return out, true
}

func parseBatteryPacket(kind string) PacketParser {
	return func(model ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			pairs, ok := parseCountedPairs(pkt.Payload)
			if !ok {
				return ParsedPacket{
					Kind:    kind,
					Summary: fmt.Sprintf("%s: % x", kind, pkt.Payload),
					Warnings: []string{
						"payload does not match [count][id,value] battery layout",
					},
				}
			}

			data := map[string]Battery{}
			for _, p := range pairs {
				name, known := partNames[p[0]]
				if !known {
					name = fmt.Sprintf("id_%d", p[0])
				}
				data[name] = Battery{
					Percent:  int(p[1] & 0x7F),
					Charging: p[1]&0x80 != 0,
				}
			}
			data, warnings := NormalizeBatteryForModel(data, model)

			return ParsedPacket{
				Kind:      kind,
				Summary:   fmt.Sprintf("%s: %s", kind, FormatBattery(data)),
				Batteries: data,
				Warnings:  warnings,
			}
		}
	}
}

// parseStatusPacket decodes GET_EARPHONE_STATUS (rsp 0x400A): [count][id,flags]*.
// Flag bits per EarphoneStatus: bit0=in_case/case_open, bit2=in_ear, bit7=connected.
func parseStatusPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			pairs, ok := parseCountedPairs(pkt.Payload)
			if !ok {
				return ParsedPacket{
					Kind:    kind,
					Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), pkt.Payload),
				}
			}

			parts := make([]string, 0, len(pairs))
			for _, p := range pairs {
				name, known := partNames[p[0]]
				if !known {
					name = fmt.Sprintf("id_%d", p[0])
				}
				v := p[1]
				var flags []string
				if p[0] == 4 { // case: bit0 = lid open
					if v&0x01 != 0 {
						flags = append(flags, "open")
					} else {
						flags = append(flags, "closed")
					}
				} else {
					if v&0x04 != 0 {
						flags = append(flags, "in_ear")
					} else if v&0x01 != 0 {
						flags = append(flags, "in_case")
					} else {
						flags = append(flags, "out")
					}
					if v&0x80 != 0 {
						flags = append(flags, "connected")
					}
				}
				parts = append(parts, fmt.Sprintf("%s[%s]", name, strings.Join(flags, ",")))
			}

			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: %s", kind, strings.Join(parts, " ")),
			}
		}
	}
}

var configTypeLabels = map[int]string{
	1: "hardware",
	2: "firmware",
	3: "firmware_backup",
	4: "serial",
	5: "manufacture_date",
	6: "bt_address",
}

// parseConfigPacket decodes GET_REMOTE_CONFIGURATION (rsp 0x4006): a leading
// count byte followed by newline-separated "device,type,value" CSV records.
func parseConfigPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			if len(pkt.Payload) < 1 {
				return ParsedPacket{Kind: kind, Summary: kind + ": (empty)"}
			}

			text := strings.TrimSpace(string(pkt.Payload[1:]))
			grouped := map[byte][]string{}
			order := []byte{}
			for _, line := range strings.Split(text, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				fields := strings.SplitN(line, ",", 3)
				if len(fields) != 3 {
					continue
				}
				dev, err1 := strconv.Atoi(fields[0])
				typ, err2 := strconv.Atoi(fields[1])
				val := fields[2]
				if err1 != nil || err2 != nil || val == "" {
					continue
				}
				label, ok := configTypeLabels[typ]
				if !ok {
					label = fmt.Sprintf("type_%d", typ)
				}
				id := byte(dev)
				if _, seen := grouped[id]; !seen {
					order = append(order, id)
				}
				grouped[id] = append(grouped[id], fmt.Sprintf("%s=%s", label, val))
			}

			parts := make([]string, 0, len(order))
			for _, id := range order {
				name, known := partNames[id]
				if !known {
					name = fmt.Sprintf("id_%d", id)
				}
				parts = append(parts, fmt.Sprintf("%s{%s}", name, strings.Join(grouped[id], ", ")))
			}

			return ParsedPacket{
				Kind:    kind,
				Text:    text,
				Summary: fmt.Sprintf("%s: %s", kind, strings.Join(parts, " ")),
			}
		}
	}
}

func parseSupportedFeaturePacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: dual_list=%t payload=% x", kind, SupportedFeatureDualList(pkt.Payload), pkt.Payload),
			}
		}
	}
}

// SupportedFeatureDualList uses payload[1] bit 6 as GET_DUAL_DEVICE_LIST gate.
func SupportedFeatureDualList(payload []byte) bool {
	return len(payload) > 1 && payload[1]&0x40 != 0
}

func parseRawPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), pkt.Payload),
			}
		}
	}
}

func parseTextPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			return ParsedPacket{
				Kind:    kind,
				Text:    string(pkt.Payload),
				Summary: fmt.Sprintf("%s raw=%q hex=% x", kind, pkt.Payload, pkt.Payload),
			}
		}
	}
}

// ANC (GET_CURRENT_NOISE_REDUCTION, rsp 0x401E) is a list of 3-byte triples
// (type, value, none); type 1 = mode/tab, type 2 = last manual level.
var ancModeLabels = map[byte]string{
	0:   "off",
	1:   "high",
	2:   "mid",
	3:   "low",
	4:   "adaptive",
	5:   "off",
	7:   "transparency",
	254: "transparency",
}

var ancLevelLabels = map[byte]string{
	1: "high",
	2: "mid",
	3: "low",
	4: "adaptive",
}

func ancModeLabel(v byte) string {
	if label, ok := ancModeLabels[v]; ok {
		return label
	}
	if v >= 1 && v <= 127 {
		return fmt.Sprintf("anc(level=%d)", v)
	}
	return fmt.Sprintf("mode_%d", v)
}

func parseANCPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			p := pkt.Payload
			if len(p) < 3 || len(p)%3 != 0 {
				return ParsedPacket{
					Kind:    kind,
					Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), p),
				}
			}
			mode, level := "", ""
			for i := 0; i+2 < len(p); i += 3 {
				switch p[i] {
				case 1:
					mode = ancModeLabel(p[i+1])
				case 2:
					if label, ok := ancLevelLabels[p[i+1]]; ok {
						level = label
					} else {
						level = fmt.Sprintf("level_%d", p[i+1])
					}
				}
			}
			summary := fmt.Sprintf("%s: mode=%s", kind, mode)
			if level != "" {
				summary += " last_level=" + level
			}
			return ParsedPacket{Kind: kind, Summary: summary}
		}
	}
}

// EQ preset (GET_EQ_MODE, rsp 0x401F): single byte preset index.
var eqPresetLabels = map[byte]string{
	0: "balanced",
	1: "voice",
	2: "more_treble",
	3: "more_bass",
	4: "custom",
}

func parseEQPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			if len(pkt.Payload) < 1 {
				return ParsedPacket{Kind: kind, Summary: kind + ": (empty)"}
			}
			v := pkt.Payload[0]
			label, ok := eqPresetLabels[v]
			if !ok {
				label = fmt.Sprintf("preset_%d", v)
			}
			return ParsedPacket{Kind: kind, Summary: fmt.Sprintf("%s: %s", kind, label)}
		}
	}
}

// Spatial audio (GET_SPATIAL_AUDIO, rsp 0x404F): [spatial, head_tracking] flags.
func parseSpatialPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			p := pkt.Payload
			if len(p) < 2 {
				return ParsedPacket{Kind: kind, Summary: fmt.Sprintf("%s: payload=% x", kind, p)}
			}
			onoff := func(b byte) string {
				if b != 0 {
					return "on"
				}
				return "off"
			}
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: spatial=%s head_tracking=%s", kind, onoff(p[0]), onoff(p[1])),
			}
		}
	}
}

func parseLagPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			if len(pkt.Payload) < 1 {
				return ParsedPacket{
					Kind:    kind,
					Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), pkt.Payload),
				}
			}
			mode := pkt.Payload[0]
			if len(pkt.Payload) >= 2 && pkt.Payload[0] == 0 {
				mode = pkt.Payload[1]
			}
			state := "on"
			if mode == 2 {
				state = "off"
			}
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: low_latency=%s mode=%d", kind, state, mode),
			}
		}
	}
}

// parseSetAckPacket decodes the 0x7XXX acknowledgement for a 0xFXXX SET command.
// The first payload byte is a status code where 0x00 means success.
func parseSetAckPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			if len(pkt.Payload) == 0 {
				return ParsedPacket{
					Kind:    kind,
					Summary: fmt.Sprintf("%s: ok", kind),
				}
			}
			status := pkt.Payload[0]
			result := "ok"
			if status != 0 {
				result = fmt.Sprintf("error=0x%02x", status)
			}
			if len(pkt.Payload) > 1 {
				return ParsedPacket{
					Kind:    kind,
					Summary: fmt.Sprintf("%s: %s payload=% x", kind, result, pkt.Payload),
				}
			}
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: %s", kind, result),
			}
		}
	}
}

func parseDualPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			if len(pkt.Payload) < 1 {
				return ParsedPacket{
					Kind:    kind,
					Summary: fmt.Sprintf("%s: rsp=%02x payload=% x", kind, pkt.RspCode(), pkt.Payload),
				}
			}
			enabled := pkt.Payload[0] == 1
			state := "off"
			if enabled {
				state = "on"
			}
			return ParsedPacket{
				Kind:    kind,
				Summary: fmt.Sprintf("%s: dual=%s value=%d", kind, state, pkt.Payload[0]),
			}
		}
	}
}

func parseDualDeviceListPacket(kind string) PacketParser {
	return func(ModelInfo) func(Packet) ParsedPacket {
		return func(pkt Packet) ParsedPacket {
			if len(pkt.Payload) == 0 {
				return ParsedPacket{Kind: kind, Summary: kind + ": empty"}
			}
			list, err := ParseDualDeviceListPayload(pkt.Payload)
			if err != nil {
				return ParsedPacket{
					Kind:     kind,
					Summary:  fmt.Sprintf("%s: payload=% x", kind, pkt.Payload),
					Warnings: []string{err.Error()},
				}
			}
			summaries := FormatDualDeviceSummaries(list.Devices)
			if len(summaries) == 0 {
				return ParsedPacket{
					Kind:     kind,
					Summary:  fmt.Sprintf("%s: total=%d current=%d count=%d payload=% x", kind, list.Total, list.Current, len(list.Devices), pkt.Payload),
					DualList: &list,
				}
			}
			return ParsedPacket{
				Kind:     kind,
				Summary:  fmt.Sprintf("%s: total=%d current=%d count=%d %s", kind, list.Total, list.Current, len(list.Devices), strings.Join(summaries, " | ")),
				DualList: &list,
			}
		}
	}
}

// ParseDualDeviceListPayload decodes the dual device list response body.
func ParseDualDeviceListPayload(payload []byte) (DualDeviceList, error) {
	if len(payload) < 3 {
		return DualDeviceList{}, fmt.Errorf("dual device list payload too short: %d bytes", len(payload))
	}
	list := DualDeviceList{
		Total:      payload[0],
		Current:    payload[1],
		RawPayload: append([]byte(nil), payload...),
	}
	count := int(payload[2])
	list.Devices = parseDualDeviceRecords(payload[3:], count)
	return list, nil
}

func parseDualDeviceRecords(payload []byte, count int) []DualDevice {
	devices := make([]DualDevice, 0, count)
	off := 0
	for i := 0; i < count && off+7 <= len(payload); i++ {
		state := payload[off]
		macBytes := payload[off+1 : off+7]
		off += 7

		name := ""
		if off < len(payload) {
			nameLen := int(payload[off] & 0x7F)
			remainingDevices := count - i - 1
			if nameLen <= 31 && off+1+nameLen <= len(payload) && len(payload)-(off+1+nameLen) >= remainingDevices*7 {
				name = cleanDualDeviceName(payload[off+1 : off+1+nameLen])
				off += 1 + nameLen
			} else if off+31 <= len(payload) && len(payload)-(off+31) >= remainingDevices*7 {
				name = cleanDualDeviceName(payload[off : off+31])
				off += 31
			}
		}

		devices = append(devices, DualDevice{
			MAC:       formatDualMAC(macBytes),
			Name:      name,
			Connected: state&0x0F != 0,
			Owner:     state&0xF0 != 0,
			RawState:  state,
		})
	}
	return devices
}

func FormatDualDeviceSummaries(devices []DualDevice) []string {
	items := make([]string, 0, len(devices))
	for _, dev := range devices {
		connected := "disconnected"
		if dev.Connected {
			connected = "connected"
		}
		owner := "other"
		if dev.Owner {
			owner = "owner"
		}
		if dev.Name != "" {
			items = append(items, fmt.Sprintf("%s %q [%s,%s]", dev.MAC, dev.Name, connected, owner))
		} else {
			items = append(items, fmt.Sprintf("%s [%s,%s]", dev.MAC, connected, owner))
		}
	}
	return items
}

// ParseDualMAC accepts AA:BB:CC:DD:EE:FF, AA-BB-..., or AABBCCDDEEFF.
func ParseDualMAC(raw string) ([6]byte, error) {
	var out [6]byte
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 12 {
		return out, fmt.Errorf("invalid MAC %q: want 6 bytes in AA:BB:CC:DD:EE:FF form", raw)
	}
	for i := 0; i < 6; i++ {
		part, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return out, fmt.Errorf("invalid MAC %q: %w", raw, err)
		}
		out[i] = byte(part)
	}
	return out, nil
}

// BuildDualConnectPayload builds the SET_CONNECT_DEVICE (0xF01B) payload.
func BuildDualConnectPayload(connect bool, mac string) ([]byte, []string, error) {
	addr, err := ParseDualMAC(mac)
	if err != nil {
		return nil, nil, err
	}
	flag := byte(0)
	if connect {
		flag = 1
	}
	payload := make([]byte, 7)
	payload[0] = flag
	copy(payload[1:], addr[:])
	return payload, nil, nil
}

func cleanDualDeviceName(raw []byte) string {
	end := len(raw)
	for end > 0 && (raw[end-1] == 0 || raw[end-1] == ' ') {
		end--
	}
	return strings.TrimSpace(string(raw[:end]))
}

func formatDualMAC(raw []byte) string {
	if len(raw) < 6 {
		return fmt.Sprintf("% x", raw)
	}
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", raw[0], raw[1], raw[2], raw[3], raw[4], raw[5])
}

func ParsePacket(pkt Packet, model ModelInfo) ParsedPacket {
	for _, key := range []uint16{pkt.ParserKey(), pkt.Cmd, pkt.ResponseCmd()} {
		if parser, ok := packetParsers[key]; ok {
			return parser(model)(pkt)
		}
	}
	return parseUnknownPacket(pkt, model)
}

func parseUnknownPacket(pkt Packet, model ModelInfo) ParsedPacket {
	if text, ok := printableText(pkt.Payload); ok {
		return ParsedPacket{
			Kind:    "unknown_text",
			Text:    text,
			Summary: fmt.Sprintf("unknown_text: rsp=%02x cmd=%04x text=%q hex=% x", pkt.RspCode(), pkt.Cmd, text, pkt.Payload),
		}
	}

	if data := ParsePairs(pkt.Payload); data != nil {
		data, warnings := NormalizeBatteryForModel(data, model)
		warnings = append(warnings, "unknown command has payload shaped like battery id/value pairs")

		return ParsedPacket{
			Kind:      "unknown_battery_pairs",
			Summary:   fmt.Sprintf("unknown_battery_pairs: rsp=%02x cmd=%04x %s", pkt.RspCode(), pkt.Cmd, FormatBattery(data)),
			Batteries: data,
			Warnings:  warnings,
		}
	}

	bitView := ""
	if len(pkt.Payload) > 0 && len(pkt.Payload) <= 8 {
		parts := make([]string, 0, len(pkt.Payload))
		for _, b := range pkt.Payload {
			parts = append(parts, fmt.Sprintf("%08b", b))
		}
		bitView = " bits=" + strings.Join(parts, " ")
	}

	return ParsedPacket{
		Kind:    "unknown",
		Summary: fmt.Sprintf("unknown: rsp=%02x cmd=%04x payload=% x%s", pkt.RspCode(), pkt.Cmd, pkt.Payload, bitView),
	}
}

func printableText(payload []byte) (string, bool) {
	if len(payload) == 0 || !utf8.Valid(payload) {
		return "", false
	}

	text := payload
	for len(text) > 0 {
		r, size := utf8.DecodeRune(text)
		if r == utf8.RuneError && size == 1 {
			return "", false
		}
		if r != '\n' && r != '\r' && r != '\t' && !unicode.IsPrint(r) {
			return "", false
		}
		text = text[size:]
	}

	return string(payload), true
}

func ReadPacket(r io.Reader) ([]byte, error) {
	for {
		b := make([]byte, 1)
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, err
		}
		if b[0] == SOF {
			buf := []byte{SOF}
			headerRest, err := readExact(r, 7)
			if err != nil {
				return nil, err
			}
			buf = append(buf, headerRest...)

			length := int(getUint16LE(buf, 5))
			if length > 4096 {
				return nil, fmt.Errorf("payload length too large: %d", length)
			}

			payload, err := readExact(r, length)
			if err != nil {
				return nil, err
			}
			buf = append(buf, payload...)

			if getUint16LE(buf, 1)&ControlCRC != 0 {
				crc, err := readExact(r, 2)
				if err != nil {
					return nil, err
				}
				buf = append(buf, crc...)
			}

			return buf, nil
		}
	}
}

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

func FeatureCommandPacket(fields []string, allowUnsafe bool, model ModelInfo) (Packet, []string, error) {
	if len(fields) == 0 {
		return Packet{}, nil, fmt.Errorf("missing feature command")
	}

	spec, ok := featureCommands[strings.ToLower(fields[0])]
	if !ok {
		return Packet{}, nil, fmt.Errorf("unknown feature command %q", fields[0])
	}

	action := "get"
	args := fields[1:]
	if len(args) > 0 {
		action = strings.ToLower(args[0])
		args = args[1:]
	}

	var warnings []string
	if !ModelSupportsFeature(model, spec.Feature) {
		warnings = append(warnings, fmt.Sprintf("%s is not listed for model %s tier %s", spec.Name, model.Codename, model.Tier))
	}

	switch action {
	case "get":
		if len(args) != 0 {
			return Packet{}, warnings, fmt.Errorf("usage: %s", spec.Usage)
		}

		return Packet{
			Cmd:     spec.GetCommand,
			Payload: append([]byte(nil), spec.GetPayload...),
		}, warnings, nil
	case "list":
		if spec.Feature != "dual" || len(args) != 0 {
			return Packet{}, warnings, fmt.Errorf("usage: %s", spec.Usage)
		}
		return Packet{Cmd: CmdGetDualDeviceList, Payload: []byte{0}}, warnings, nil
	case "connect", "disconnect":
		if !allowUnsafe && !spec.SafeSet {
			return Packet{}, warnings, fmt.Errorf("dual %s writes to the device; re-run with --unsafe to allow it", action)
		}
		if spec.Feature != "dual" {
			return Packet{}, warnings, fmt.Errorf("usage: %s", spec.Usage)
		}
		if len(args) != 1 {
			return Packet{}, warnings, fmt.Errorf("usage: dual %s <mac>", action)
		}
		payload, connectWarnings, err := BuildDualConnectPayload(action == "connect", args[0])
		if err != nil {
			return Packet{}, warnings, err
		}
		warnings = append(warnings, connectWarnings...)
		return Packet{
			Cmd:     CmdSetConnectDevice,
			Payload: payload,
		}, warnings, nil
	case "set":
		if !allowUnsafe && !spec.SafeSet {
			return Packet{}, warnings, fmt.Errorf("%s set writes to the device; re-run with --unsafe to allow it", spec.Name)
		}
		if model.Codename == "" {
			return Packet{}, warnings, fmt.Errorf("%s set requires a known model; pass --model or connect through discovery so it can be auto-detected", spec.Name)
		}

		payload, err := spec.BuildSetPayload(model, args)
		if err != nil {
			return Packet{}, warnings, err
		}

		return Packet{
			Cmd:     spec.SetCommand,
			Payload: payload,
		}, warnings, nil
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
