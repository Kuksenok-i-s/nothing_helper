package presenter

import (
	"strings"
	"testing"

	"tws_manager/internal/spp"
)

func TestToggleFeatures(t *testing.T) {
	model, ok := spp.ResolveModelInfo("EarTwos")
	if !ok {
		t.Fatal("EarTwos model not found")
	}
	cmds := BuildCommands(model, nil, true)

	toggles := ToggleFeatures(cmds)
	want := map[string]bool{"lag": true, "spatial": true, "dual": true}
	for _, tf := range toggles {
		if !want[tf.Feature] {
			t.Errorf("unexpected toggle feature %q", tf.Feature)
		}
		if len(tf.OnFields) != 3 || len(tf.OffFields) != 3 {
			t.Errorf("%s missing on/off fields: on=%v off=%v", tf.Feature, tf.OnFields, tf.OffFields)
		}
		delete(want, tf.Feature)
	}
	if len(want) != 0 {
		t.Errorf("missing toggles: %v", want)
	}

	// Toggle SET commands must be classified as such and excluded elsewhere.
	for _, c := range cmds {
		if len(c.Fields) == 3 && c.Fields[1] == "set" &&
			(c.Fields[2] == "on" || c.Fields[2] == "off") {
			switch c.Fields[0] {
			case "lag", "spatial", "dual":
				if !IsToggleSetCommand(c) {
					t.Errorf("%q should be a toggle set command", c.Title)
				}
			}
		}
	}
}

func TestToggleStateOn(t *testing.T) {
	cases := []struct {
		feature, value string
		want           bool
	}{
		{"lag", "low_latency=on mode=1", true},
		{"lag", "low_latency=off mode=2", false},
		{"spatial", "spatial=on head_tracking=off", true},
		{"spatial", "spatial=off head_tracking=off", false},
		{"dual", "dual=on value=1", true},
		{"dual", "dual=off value=0", false},
	}
	for _, c := range cases {
		if got := ToggleStateOn(c.feature, c.value); got != c.want {
			t.Errorf("ToggleStateOn(%q,%q)=%v want %v", c.feature, c.value, got, c.want)
		}
	}
}

func TestBuildCommandsIncludesBattery(t *testing.T) {
	cmds := BuildCommands(spp.DefaultModel(), nil, false)
	found := false
	for _, c := range cmds {
		if strings.Contains(c.Title, "battery") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("battery command missing")
	}
}

func TestNeedsUnsafeConfirmation(t *testing.T) {
	if !NeedsUnsafeConfirmation(Command{Fields: []string{"anc", "set", "off"}}) {
		t.Fatal("expected unsafe confirmation")
	}
	if NeedsUnsafeConfirmation(Command{Fields: []string{"anc", "get"}}) {
		t.Fatal("get should not need confirmation")
	}
}

func TestBuildCommandsIncludesLowLatencySetWithoutUnsafe(t *testing.T) {
	model, ok := spp.ResolveModelInfo("EarThree")
	if !ok {
		t.Fatal("EarThree model not found")
	}
	cmds := BuildCommands(model, nil, false)
	var get, set bool
	for _, c := range cmds {
		if len(c.Fields) >= 2 && c.Fields[0] == "lag" && c.Fields[1] == "get" {
			get = true
		}
		if len(c.Fields) >= 2 && c.Fields[0] == "lag" && c.Fields[1] == "set" {
			set = true
		}
	}
	if !get {
		t.Fatal("low latency GET command missing")
	}
	if !set {
		t.Fatal("low latency SET command missing")
	}
}

func TestBuildCommandsIncludesWriteControlsForDefaultModel(t *testing.T) {
	cmds := BuildCommands(spp.DefaultModel(), nil, false)
	var ancSet bool
	for _, c := range cmds {
		if len(c.Fields) >= 2 && c.Fields[0] == "anc" && c.Fields[1] == "set" && c.SafeSet {
			ancSet = true
			break
		}
	}
	if !ancSet {
		t.Fatal("default model should expose safe UI write controls")
	}
	if toggles := ToggleFeatures(cmds); len(toggles) == 0 {
		t.Fatal("default model should expose write toggles")
	}
}

func TestBuildCommandsIncludesScanOnlyWithUnsafe(t *testing.T) {
	model, ok := spp.ResolveModelInfo("EarThree")
	if !ok {
		t.Fatal("EarThree model not found")
	}
	for _, c := range BuildCommands(model, nil, false) {
		if IsScanCommand(c) {
			t.Fatal("raw scan should be hidden without unsafe")
		}
	}
	cmds := BuildCommands(model, nil, true)
	for _, c := range cmds {
		if IsScanCommand(c) {
			return
		}
	}
	t.Fatal("raw scan missing with unsafe enabled")
}

func TestCommandClassification(t *testing.T) {
	get := Command{Title: "Info: battery", Cmd: spp.CmdGetBattery}
	set := Command{Title: "SET: anc off", Fields: []string{"anc", "set", "off"}}
	dual := Command{Title: "Dual: connect X", Fields: []string{"dual", "connect", "AA:BB:CC:DD:EE:FF"}}
	scan := Command{Title: "Advanced: raw scan", Advanced: true}

	if !IsGetCommand(get) || IsSetCommand(get) || IsDualAction(get) {
		t.Fatal("battery should classify as GET only")
	}
	if !IsSetCommand(set) || IsGetCommand(set) {
		t.Fatal("anc off should classify as SET only")
	}
	if !IsDualAction(dual) || IsGetCommand(dual) || IsSetCommand(dual) {
		t.Fatal("dual connect should classify as dual action only")
	}
	if IsGetCommand(scan) || IsSetCommand(scan) {
		t.Fatal("advanced scan should be neither GET nor SET")
	}
}
