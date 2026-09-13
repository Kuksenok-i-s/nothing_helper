# Ear (3): Super Mic / Walkie Talkie research

Static analysis of Nothing X 3.5.3, performed 2026-09-13. No hardware
commands were sent. Synthetic parser tests do not prove firmware behavior.

## Product behavior

[Nothing's description](https://support.nothing.tech/hc/en-us/articles/38937909284881-What-is-Walkie-Talkie-Mode)
describes push-to-talk during calls: microphones remain muted until the TALK
button on the charging case is pressed. This is a call microphone mode;
there is no evidence here of a separate peer-to-peer radio transport.

## Commands recovered from the app

- Read Super Mic: `0xC05E`, empty payload; response `0x405E`.
- Set Super Mic: `0xF05F`, one byte: `01` enabled, `00` disabled.
- Read Walkie Talkie: `0xC060`, empty payload; response `0x4060`.
- Set Walkie Talkie: `0xF061`, one byte: `01` enabled, `00` disabled.
- Walkie Talkie notification: `0xE01A`, first byte is the state.

The app reads the first payload integer, interpreting 1 as true. Empty data
does not update state. Our passive parser additionally rejects values other
than 0/1 and payloads longer than one byte until hardware captures clarify
extensions. ACK codes/payloads for these SETs have not been individually
verified; do not treat ACK payloads as the feature state.

## Ordering and desktop integration

`switchWalkTalk(true)` calls `setSuperMicEnable(true)` when the recorded
Super Mic state is not true, then `setWalkieTalkieMode(true)`.
`switchWalkTalk(false)` only sets Walkie Talkie false.
`switchSuperMic(false)` also sets Walkie Talkie false when recorded as on.
The inspected wrapper does not await an acknowledgement between the SETs.

Android's `NtSupperMicApi` observes Bluetooth audio output devices and bridges
toggles to Flutter. `WalkieTalkieService` provides a foreground notification.
No PCM transport was found in these classes. This is evidence for an audio
path separate from SPP control, not proof of its exact Bluetooth profile.

Hardware validation still needed:

1. Capture reads and toggles in Nothing X on Ear (3), recording firmware.
2. Confirm model/capability gating; the app uses configurable
   `isSupportSuperMic` and `isSupportWalkieTalkieMode` predicates, so do not
   enable this for every device using EarTwosProtocol.
3. Verify enabling, disabling, TALK press/release and reconnection while a
   call microphone is selected on Linux and macOS. Reading "enabled" alone
   does not prove that case audio reaches the desktop application.
4. Validate partial-failure behavior against firmware. The GUI now serializes
   Super Mic then Walkie Talkie writes, reads back each state and stops on
   timeout/error. Disabling Walkie Talkie leaves Super Mic enabled.

`0xE018` (recording) and SkyWalk/Essential Space are not established as
Walkie Talkie events. Do not conflate them. The APK dispatch also treats
`0xE019` as Audiodo; this differs from the existing lag-event mapping in this
project and needs a separate model-aware investigation.

## Reproduction evidence

Inputs from the user's local APK directory:

- `base.apk` SHA-256:
  `44a70dbc2c19533387c6098143c8bd1669b79fa49bf5835295302d2acacb78de`
- `lib/arm64-v8a/libapp.so` from `split_config.arm64_v8a.apk` SHA-256:
  `c1bb657a7225c5c6e84b394c06135b14dc93f0c8e46bb6e02f9d863efcb15151`
- Dart 3.9.2, ARM64 compressed pointers; inspected with jadx and
  [Blutter](https://github.com/worawit/blutter).

Named ARM64 AOT locations, not raw packet captures:

- `ear/ear_helper.dart`: `getWalkieTalkieMode` at `0x15f4640`,
  `getSuperMicEnable` at `0x15f48ac`, `setWalkieTalkieMode` at `0x16051ac`,
  `setSuperMicEnable` at `0x1605adc`.
- `EarProtocol.updateData` normalizes received command with `| 0x8000`
  at `0x28b1d08`; Super Mic branch `0x28b35d0`, Walkie response branch
  `0x28b3814`, shared Walkie response/event parser `0x28b461c`.
  Event dispatch compares `0xE019` and `0xE01A` at `0x28b43c8` onward.
- `core/bluetooth_helper.dart`: `switchSuperMic` at `0x1604f98`,
  `switchWalkTalk` at `0x16065e0`.

Only derived protocol facts are kept here; proprietary decompiled sources
remain outside the repository. Current implementation includes an Ear (3)-only GUI toggle, initial and periodic
queries, and strict one-byte validation at the session write boundary (including
with --unsafe). Hardware behavior remains unverified.


## Linux hardware investigation, 2026-09-13

Session `captures/session_2026-09-13_14-56-49.ndjson` confirms the control
path on the connected EarThree: SET `F05F`/`F061` receive empty ACKs
`705F`/`7061`, followed by `405E`/`4060` reporting enabled=true.
`E01A` reporting false was also observed. These are actual device responses,
not synthetic fixtures; the firmware version was not recorded in this note.

Audio observations on PipeWire 1.6.8:

- Initial card profile A2DP/LDAC has zero hardware capture sources. A
  persistent `bluez_input.<colon-separated-MAC>` loopback source nevertheless
  appears RUNNING; that alone does not prove a working microphone.
- A 15-second parec test after selecting HFP/mSBC yielded 14 seconds of
  all-zero PCM. A second test using media.role=phone yielded no PCM frames.
- During the latter test the virtual source was RUNNING while the physical
  `bluez_input.<underscore-separated-MAC>.0` node was SUSPENDED.
- Direct pw-record targeting the physical node activated it, but delivered
  only 0.6169375 seconds of zero PCM over approximately 12 seconds.
- WirePlumber repeatedly reported Bluetooth audio transport failures. Kernel
  log at 15:03:45 reported `SCO packet for unknown connection handle 513`.

Therefore command acceptance is established, but usable audio capture is
not. Failure is not explained solely by the GUI switch. Transport failure,
virtual-source routing and the device's call-state behavior still need
separate control tests (Walkie Talkie disabled, ordinary earbud microphone,
then Super Mic). A direct CVSD test delivered 2.705875 seconds over a 10-second window,
with peak 22/32768 (about -63 dBFS). This tiny signal does not establish
intelligible speech or successful push-to-talk operation.
Temporary audio profiles were restored to A2DP after direct-source tests.


Control recording at 15:08 with confirmed Walkie Talkie=false before and
at the end of capture produced 0.659625 seconds of all-zero PCM during a
15-second physical-HFP-input test. Super Mic remained enabled, so this
isolates the Walkie Talkie switch but not all case-microphone behavior.
WirePlumber also logged repeated `wp_properties_get: self != NULL`
assertions. An earlier attempt was excluded because the user accidentally
re-enabled the mode before recording.


After restarting WirePlumber, the Walkie-disabled control test delivered
9.9609375 seconds over 10 seconds, peak 9871/32768 and 87256 nonzero samples.
The subsequent user-assisted TALK test delivered 14.931625 seconds,
peak 19003/32768. Seconds 0–2 were effectively silent, seconds 3–14 had
strong audio. However, the device sent E01A=false at 15:13:16.320824905,
confirmed by GET 4060 at 15:13:19, while the pre-test GET was true.
There was no companion F061 write in that interval. Thus audio capture is
restored, but hold-to-talk muting is NOT confirmed. A call/audio-profile
transition resetting mode is a hypothesis; the next test should establish
HFP and keep the capture stream open before enabling Walkie Talkie, then
compare press/release while checking that the mode stays enabled.


## Continuous-stream test at 15:21

A 30-second capture with HFP kept open delivered 29.9503125 seconds.
Capture began at 15:21:24.839. The device sent E01A=false at 15:21:27.242.
The user enabled Walkie Talkie after stream startup: companion F061 at
15:21:33.688, GET response true at 15:21:34.251 and still true at
15:21:39.847 and 15:21:50.070. No further false event occurred during capture.
There was strong audio interspersed with entirely zero intervals (seconds
16, 20–21, 24–27). This is consistent with press-to-talk gating, but button
press/release timestamps were not independently recorded.

This establishes a working capture with the mode remaining enabled when
set AFTER opening the HFP stream. Starting the stream while the mode was
already enabled caused a false event in this test. Production behavior
should account for audio-session transitions rather than continually
forcing the mode back on. The original A2DP profile was restored.


## GUI activation guard

The controller now checks an active capture path before either enabling
Super Mic or enabling Walkie Talkie, and checks again between the commands.
Linux uses the physical running PipeWire HFP Audio/Source node for the exact
MAC, rejecting the persistent virtual A2DP input. macOS queries CoreAudio
for a matching Bluetooth UID with input streams, running I/O and a call
sample rate. The macOS heuristic still needs an Ear (3) hardware check.

An inactive or unverifiable path gives an instruction to start a call or
recording with the Ear (3) microphone first. Disabling Walkie Talkie is not
subject to this guard. The application does not switch profiles or open an
unrequested recording as part of enabling the toggle.
