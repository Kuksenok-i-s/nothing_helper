//go:build darwin

// spp_spike is a minimal macOS hardware spike: open RFCOMM channel 15,
// send GET_BATTERY, and parse the response using internal/spp.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"tws_manager/internal/bt"
	"tws_manager/internal/security"
	"tws_manager/internal/spp"
)

func main() {
	mac := flag.String("addr", "", "paired device MAC (required)")
	channel := flag.Int("channel", 15, "RFCOMM channel")
	timeout := flag.Duration("timeout", time.Second, "read timeout")
	openOnly := flag.Bool("open-only", false, "open RFCOMM and exit (no GET_BATTERY)")
	exact := flag.Bool("exact", false, "try only the requested channel (no fallback to 15)")
	flag.Parse()

	if *mac == "" {
		fmt.Fprintln(os.Stderr, "usage: spp_spike --addr AA:BB:CC:DD:EE:FF [--channel 15] [--open-only] [--exact]")
		os.Exit(2)
	}
	normMAC, err := security.NormalizeMAC(*mac)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid MAC: %v\n", err)
		os.Exit(1)
	}
	if err := security.ValidateChannel(*channel); err != nil {
		fmt.Fprintf(os.Stderr, "invalid channel: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "opening RFCOMM %s channel %d...\n", normMAC, *channel)
	var transport bt.Transport
	var usedCh int
	progress := func(step string) {
		fmt.Fprintf(os.Stderr, "  %s\n", step)
	}
	if *exact {
		transport, usedCh, err = bt.OpenTransportExact(normMAC, *channel, progress)
	} else {
		transport, usedCh, err = bt.OpenTransport(normMAC, normMAC, *channel, progress)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "open failed: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close()
	fmt.Fprintf(os.Stderr, "opened %s on channel %d\n", transport.String(), usedCh)

	if *openOnly {
		fmt.Fprintln(os.Stderr, "open PASS")
		return
	}

	pkt := spp.Packet{Cmd: spp.CmdGetBattery}
	pkt.FSN = spp.NextFSN()
	pkt.FixedFSN = true
	raw := pkt.MarshalBinary()
	fmt.Fprintf(os.Stderr, "TX GET_BATTERY (%d bytes)\n", len(raw))
	if _, err := transport.Write(raw); err != nil {
		fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		os.Exit(1)
	}

	deadline := time.Now().Add(*timeout)
	var buf []byte
	for time.Now().Before(deadline) {
		chunk := make([]byte, 512)
		n, err := transport.Read(chunk)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			os.Exit(1)
		}
		if n == 0 {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		buf = append(buf, chunk[:n]...)
		decoded, err := spp.DecodePacket(buf)
		if err != nil {
			if strings.Contains(err.Error(), "short") || strings.Contains(err.Error(), "waiting") {
				continue
			}
			fmt.Fprintf(os.Stderr, "decode error: %v\n", err)
			os.Exit(1)
		}
		parsed := spp.ParsePacket(decoded, spp.DefaultModel())
		fmt.Printf("RX cmd=%04x summary=%s\n", decoded.Cmd, parsed.Summary)
		for side, b := range parsed.Batteries {
			fmt.Printf("  %s: %d%% charging=%v\n", side, b.Percent, b.Charging)
		}
		fmt.Fprintln(os.Stderr, "spike PASS")
		return
	}
	fmt.Fprintln(os.Stderr, "timeout waiting for battery response")
	os.Exit(1)
}
