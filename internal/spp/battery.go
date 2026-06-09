package spp

import "fmt"

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
	0xE005: {Name: "event_buds_battery", Kind: "battery_pairs"},
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

