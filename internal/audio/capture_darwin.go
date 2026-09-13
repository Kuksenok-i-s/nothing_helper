//go:build darwin && cgo

package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <ctype.h>
#include <string.h>

static int activeBluetoothCapture(const char *mac) {
 AudioObjectPropertyAddress a = {kAudioHardwarePropertyDevices, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
 UInt32 size = 0;
 if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &a, 0, NULL, &size) != noErr) return -1;
 AudioDeviceID *devices = malloc(size);
 if (!devices) return -1;
 if (AudioObjectGetPropertyData(kAudioObjectSystemObject, &a, 0, NULL, &size, devices) != noErr) { free(devices); return -1; }
 int found = 0;
 for (UInt32 i = 0; i < size / sizeof(AudioDeviceID); ++i) {
  AudioDeviceID d = devices[i]; UInt32 n, transport = 0, running = 0; Float64 rate = 0;
  a.mSelector = kAudioDevicePropertyTransportType; a.mScope = kAudioObjectPropertyScopeGlobal; n = sizeof(transport);
  if (AudioObjectGetPropertyData(d, &a, 0, NULL, &n, &transport) != noErr || transport != kAudioDeviceTransportTypeBluetooth) continue;
  a.mSelector = kAudioDevicePropertyDeviceUID; CFStringRef uid = NULL; n = sizeof(uid);
  if (AudioObjectGetPropertyData(d, &a, 0, NULL, &n, &uid) != noErr || !uid) continue;
  char text[512], compact[512]; int j = 0;
  Boolean ok = CFStringGetCString(uid, text, sizeof(text), kCFStringEncodingUTF8); CFRelease(uid);
  if (!ok) continue;
  for (int k = 0; text[k]; ++k) { if (text[k] != ':' && text[k] != '-') compact[j++] = tolower((unsigned char)text[k]); }
  compact[j] = 0;
  if (!strstr(compact, mac)) continue;
  a.mSelector = kAudioDevicePropertyStreams; a.mScope = kAudioObjectPropertyScopeInput; n = 0;
  if (AudioObjectGetPropertyDataSize(d, &a, 0, NULL, &n) != noErr || n == 0) continue;
  a.mSelector = kAudioDevicePropertyDeviceIsRunningSomewhere; a.mScope = kAudioObjectPropertyScopeGlobal; n = sizeof(running);
  if (AudioObjectGetPropertyData(d, &a, 0, NULL, &n, &running) != noErr || !running) continue;
  a.mSelector = kAudioDevicePropertyNominalSampleRate; n = sizeof(rate);
  // Ear (3) call codecs use narrow/wide-band rates. A2DP playback must not
  // make an otherwise idle Bluetooth microphone look ready.
  if (AudioObjectGetPropertyData(d, &a, 0, NULL, &n, &rate) == noErr && rate > 0 && rate <= 32000) { found = 1; break; }
 }
 free(devices); return found;
}
*/
import "C"

import (
	"context"
	"fmt"
	"strings"
	"nothing_helper/internal/security"
	"unsafe"
)

func HasActiveCaptureForMAC(ctx context.Context, mac string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	mac, err := security.NormalizeMAC(mac)
	if err != nil {
		return false, err
	}
	value := C.CString(strings.ToLower(strings.ReplaceAll(mac, ":", "")))
	defer C.free(unsafe.Pointer(value))
	result := C.activeBluetoothCapture(value)
	if result < 0 {
		return false, fmt.Errorf("не удалось проверить микрофон CoreAudio")
	}
	return result == 1, nil
}
