package trace

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"tws_manager/internal/spp"
)

type DeviceInfo struct {
	MAC  string `json:"mac,omitempty"`
	Name string `json:"name,omitempty"`
}

type Context struct {
	Source           string
	Trigger          string
	Device           DeviceInfo
	Model            spp.ModelInfo
	UserComment      string
	RelatedTXCommand string
}

type Event struct {
	Time             string                 `json:"time"`
	Direction        string                 `json:"direction"`
	Source           string                 `json:"source,omitempty"`
	Trigger          string                 `json:"trigger,omitempty"`
	DeviceMAC        string                 `json:"device_mac,omitempty"`
	DeviceName       string                 `json:"device_name,omitempty"`
	ModelCodename    string                 `json:"model_codename,omitempty"`
	Command          string                 `json:"command,omitempty"`
	CommandName      string                 `json:"command_name,omitempty"`
	CatalogKind      string                 `json:"catalog_kind,omitempty"`
	ParsedKind       string                 `json:"parsed_kind,omitempty"`
	Control          string                 `json:"control,omitempty"`
	Type             string                 `json:"type,omitempty"`
	DeviceType       int                    `json:"device_type,omitempty"`
	Length           int                    `json:"length,omitempty"`
	FSN              string                 `json:"fsn,omitempty"`
	HasCRC           bool                   `json:"has_crc,omitempty"`
	CRCValid         *bool                  `json:"crc_valid,omitempty"`
	MultiFrame       bool                   `json:"multi_frame,omitempty"`
	RawHex           string                 `json:"raw_hex,omitempty"`
	RawWithoutCRC    string                 `json:"raw_without_crc,omitempty"`
	CRC              string                 `json:"crc,omitempty"`
	Summary          string                 `json:"summary,omitempty"`
	Batteries        map[string]spp.Battery `json:"batteries,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	Note             string                 `json:"note,omitempty"`
	UserComment      string                 `json:"user_comment,omitempty"`
	RelatedTXCommand string                 `json:"related_tx_command,omitempty"`
	Error            string                 `json:"error,omitempty"`
}

type CRCSample struct {
	RawWithoutCRC string
	CRC           string
}

type Logger struct {
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
	samples map[string]CRCSample
	logRaw  bool
}

func NewLogger(path string, logRaw bool) (*Logger, error) {
	samples := loadCRCSamples(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{file: file, encoder: json.NewEncoder(file), samples: samples, logRaw: logRaw}, nil
}

func loadCRCSamples(path string) map[string]CRCSample {
	samples := map[string]CRCSample{}
	data, err := os.ReadFile(path)
	if err != nil {
		return samples
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.RawWithoutCRC == "" || event.CRC == "" {
			continue
		}
		key := event.RawWithoutCRC + ":" + event.CRC
		samples[key] = CRCSample{RawWithoutCRC: event.RawWithoutCRC, CRC: event.CRC}
	}
	return samples
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

func (l *Logger) LogEvent(event Event) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if event.Time == "" {
		event.Time = time.Now().Format(time.RFC3339Nano)
	}
	if event.RawHex != "" && event.RawWithoutCRC == "" && event.CRC == "" {
		if raw, err := hex.DecodeString(event.RawHex); err == nil {
			rawWithoutCRC, crc := SplitCRC(raw)
			event.RawWithoutCRC = hex.EncodeToString(rawWithoutCRC)
			event.CRC = hex.EncodeToString(crc)
		}
	}
	if l.logRaw && event.RawWithoutCRC != "" && event.CRC != "" {
		key := event.RawWithoutCRC + ":" + event.CRC
		l.samples[key] = CRCSample{RawWithoutCRC: event.RawWithoutCRC, CRC: event.CRC}
	}
	event = redactEvent(event, l.logRaw)
	if err := l.encoder.Encode(event); err != nil {
		fmt.Fprintf(os.Stderr, "trace log error: %v\n", err)
	}
}

func (l *Logger) LogTX(raw []byte, pkt spp.Packet, ctx Context) Event {
	info := spp.CommandInfoFor(pkt.Cmd)
	rawWithoutCRC, crc := SplitCRC(raw)
	event := Event{
		Direction:     "tx",
		Source:        ctx.Source,
		Trigger:       ctx.Trigger,
		DeviceMAC:     ctx.Device.MAC,
		DeviceName:    ctx.Device.Name,
		ModelCodename: ctx.Model.Codename,
		Command:       fmt.Sprintf("%04x", pkt.Cmd),
		CommandName:   info.Name,
		CatalogKind:   info.Kind,
		Control:       fmt.Sprintf("%04x", packetControl(pkt)),
		Type:          fmt.Sprintf("%02x", pkt.RspCode()),
		DeviceType:    packetDeviceType(pkt),
		Length:        len(pkt.Payload),
		FSN:           fmt.Sprintf("%02x", pkt.FSN),
		HasCRC:        packetHasCRC(pkt),
		MultiFrame:    packetMultiFrame(pkt),
		RawHex:        hex.EncodeToString(raw),
		RawWithoutCRC: hex.EncodeToString(rawWithoutCRC),
		CRC:           hex.EncodeToString(crc),
		Summary:       fmt.Sprintf("TX cmd=%s", spp.CommandLabel(pkt.Cmd)),
		UserComment:   ctx.UserComment,
	}
	l.LogEvent(event)
	return event
}

func (l *Logger) LogRX(raw []byte, pkt spp.Packet, parsed spp.ParsedPacket, decodeErr error, ctx Context) Event {
	rawWithoutCRC, crc := SplitCRC(raw)
	event := Event{
		Direction:        "rx",
		Source:           ctx.Source,
		Trigger:          ctx.Trigger,
		DeviceMAC:        ctx.Device.MAC,
		DeviceName:       ctx.Device.Name,
		ModelCodename:    ctx.Model.Codename,
		RawHex:           hex.EncodeToString(raw),
		RawWithoutCRC:    hex.EncodeToString(rawWithoutCRC),
		CRC:              hex.EncodeToString(crc),
		UserComment:      ctx.UserComment,
		RelatedTXCommand: ctx.RelatedTXCommand,
	}
	if decodeErr != nil {
		event.Error = decodeErr.Error()
		event.Summary = "decode_error: " + decodeErr.Error()
		l.LogEvent(event)
		return event
	}
	info := spp.CommandInfoFor(pkt.Cmd)
	event.Command = fmt.Sprintf("%04x", pkt.Cmd)
	event.CommandName = info.Name
	event.CatalogKind = info.Kind
	event.ParsedKind = parsed.Kind
	event.Control = fmt.Sprintf("%04x", pkt.Control)
	event.Type = fmt.Sprintf("%02x", pkt.RspCode())
	event.DeviceType = pkt.DeviceType()
	event.Length = int(pkt.Length)
	event.FSN = fmt.Sprintf("%02x", pkt.FSN)
	event.HasCRC = pkt.HasCRC()
	event.CRCValid = &pkt.CRCValid
	event.MultiFrame = pkt.MultiFrame()
	event.Summary = parsed.Summary
	event.Batteries = parsed.Batteries
	event.Warnings = parsed.Warnings
	l.LogEvent(event)
	return event
}

func packetControl(pkt spp.Packet) uint16 {
	if pkt.Control != 0 {
		return pkt.Control
	}
	return spp.ControlTXDefault
}

func packetHasCRC(pkt spp.Packet) bool {
	return packetControl(pkt)&spp.ControlCRC != 0
}

func packetMultiFrame(pkt spp.Packet) bool {
	return packetControl(pkt)&spp.ControlMultiFrame != 0
}

func packetDeviceType(pkt spp.Packet) int {
	return int((packetControl(pkt) & 0x0F00) >> 8)
}

func (l *Logger) LogNote(note string, ctx Context) {
	l.LogEvent(Event{Direction: "note", Source: "manual_note", Note: note, Summary: "note: " + note, DeviceMAC: ctx.Device.MAC, DeviceName: ctx.Device.Name, ModelCodename: ctx.Model.Codename})
}

func (l *Logger) PrintCRCSamples(w io.Writer) {
	if l == nil {
		fmt.Fprintln(w, "enable --log to collect CRC samples")
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.samples) == 0 {
		fmt.Fprintln(w, "no CRC samples collected yet")
		return
	}
	for _, sample := range l.samples {
		fmt.Fprintf(w, "%s %s\n", sample.RawWithoutCRC, sample.CRC)
	}
}

func Export(path string, events []Event, comment string, logRaw bool) error {
	if !logRaw {
		fmt.Fprintln(os.Stderr, "warning: packet export omits raw bytes (enable --log-raw to include them)")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	exportEvents := make([]Event, len(events))
	for i, event := range events {
		exportEvents[i] = redactEvent(event, logRaw)
	}
	payload := struct {
		Comment string  `json:"comment,omitempty"`
		Events  []Event `json:"events"`
	}{Comment: comment, Events: exportEvents}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func SplitCRC(raw []byte) ([]byte, []byte) {
	if len(raw) < 2 {
		return raw, nil
	}
	return raw[:len(raw)-2], raw[len(raw)-2:]
}

func redactEvent(event Event, logRaw bool) Event {
	if logRaw {
		return event
	}
	event.RawHex = ""
	event.RawWithoutCRC = ""
	event.CRC = ""
	return event
}
