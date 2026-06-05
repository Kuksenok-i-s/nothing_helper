# AGENTS: tws_manager

Этот файл - быстрый справочник для агентной работы по Go-части проекта.

## 1) Что это за проект

CLI/TUI-клиент для работы с устройствами Nothing/CMF через RFCOMM SPP:
- обнаружение Bluetooth-устройств;
- управление RFCOMM (`/dev/rfcommN`) с авто-восстановлением;
- отправка/прием SPP-пакетов;
- парсинг батареи/статуса/ANC/EQ/dual/spatial;
- логирование сессии в NDJSON + экспорт пакетов в JSON.

Точки входа:
- `cmd/tws_manager/main.go` - TUI (Bubble Tea);
- `cmd/tws_manager_gio/main.go` - Gio GUI (`-tags gio`, нужен `vulkan-headers` на Linux).

## 2) Карта Go-пакетов

- `cmd/tws_manager/main.go`
  - парсит флаги через `internal/app`;
  - поднимает `session.Session`, `tray`, `tui`;
  - stdin preflight RFCOMM (bind/discover).

- `cmd/tws_manager_gio/main.go`
  - тот же bootstrap (`internal/app`);
  - `internal/connect` для discover/bind/connect из GUI;
  - `internal/ui/gio` (build tag `gio`).

- `internal/app`
  - общая валидация флагов и bootstrap (`Runtime`).

- `internal/connect`
 - UI-friendly discover/bind/connect без stdin;
 - `AutoConnect`/`ConnectBest`/`BestCandidate` - автодискавери и reconnect-on-loss.

- `internal/notify`
 - desktop-уведомления (GNOME/freedesktop) через `gdbus`/`notify-send`, no-op без них;
 - watcher событий сессии: connect/disconnect, батарея (in-place), low-battery alerts.

- `internal/ui/presenter`
  - общий лог/статус и каталог команд для TUI и Gio.

- `internal/session`
  - центральный orchestration слой;
  - хранит текущее соединение, модель, батареи, pending TX->RX (по FSN);
  - запускает read loop, initial handshake/probe;
  - ограничивает unsafe-команды и scan.

- `internal/spp`
  - wire-формат фреймов (`frame.go`): SOF/control/cmd/len/fsn/payload/crc;
  - каталог команд + парсеры ответов/ивентов (`spp.go`);
  - резолв моделей и feature-gating по модели;
  - device safety валидации (`device_safety.go`).

- `internal/bt`
  - обертки над `bluetoothctl`/`rfcomm`/`sudo`;
  - discovery и enrich metadata;
  - bind/release/revive RFCOMM, фиксы прав доступа;
  - сохранение `devicePath -> MAC` в user config.

- `internal/trace`
  - NDJSON logger TX/RX/event;
  - redaction raw bytes по умолчанию (raw только с `--log-raw`);
  - export событий в JSON.

- `internal/security`
  - input validation: MAC, `/dev/rfcommN`, channel, writable path.

- `internal/ui/tui`
  - Bubble Tea UI: devices/control/log;
  - использует `connect.Manager` и `presenter`;
  - экспорт текущей истории пакетов.

- `internal/ui/gio`
  - Gio GUI (build tag `gio`): `app`, `config`, `state`, `theme`, `widgets`, `view`

- `internal/ui/tray`
  - systray сборка (build tag `systray`) + fallback no-op;
  - меню: статус, батарея, refresh, reconnect (`OnReconnect`), disconnect, quit;
  - требует `libayatana-appindicator` (Arch: `pacman -S libayatana-appindicator`).

## 3) Runtime поток данных

1. `main` создает `Session`.
2. `Session.Connect` открывает RFCOMM через `bt.OpenRFCOMMDevice`.
3. После connect:
   - `spp.ResetFSN()`;
   - стартует `readLoop`;
   - (опционально) `initialProbe`: protocol version -> firmware -> identity -> battery.
4. TX через `Session.Send`:
   - authorizeCommand (safe/unsafe checks);
   - MarshalBinary + write в RFCOMM;
   - trace TX и pending map по `FSN`.
5. RX через `readLoop`:
   - `spp.ReadPacket` -> `spp.DecodePacket` -> `spp.ParsePacket`;
   - correlation ответа с request через echoed FSN (fallback 0x40xx->0xC0xx);
   - публикация событий в канал + подписчики.

## 4) Критические инварианты (не ломать)

- Только `/dev/rfcommN` допустим как RFCOMM path (`internal/security`).
- Unsafe/protected команды должны блокироваться без `--unsafe`.
- Scan ограничен:
  - только `0xC0xx` GET команды;
  - `delay >= 200ms`;
  - максимум 32 команды.
- Повторный connect к уже подключенному тому же MAC - idempotent no-op.
- По умолчанию raw bytes не должны попадать в экспорт/логи без `--log-raw`.
- Для моделей со stereo source батареи применяется mapping `stereo -> case`.

## 5) Где вносить изменения

### Новая команда/парсер протокола

1. Добавить константу/метаданные в `internal/spp/spp.go`:
   - `commandCatalog`;
   - при необходимости `Cmd...`.
2. Добавить parser в `packetParsers` + функцию `parse...Packet`.
3. Если это feature-команда из UI:
   - обновить `featureCommands`;
   - при SET добавить валидацию payload (по аналогии с ANC/EQ).
4. Добавить/обновить тесты в `internal/spp/spp_test.go`.

### Изменения поведения безопасности

- Основная точка: `internal/spp/device_safety.go` и `Session.authorizeCommand`.
- Обязательно обновлять:
  - `internal/spp/device_safety_test.go`;
  - `internal/session/session_test.go` (блокировки unsafe).

### Изменения RFCOMM/подключения

- Основная точка: `internal/bt/bt.go` + `Session.Connect`.
- Покрыть:
  - recoverable ошибки open/revive;
  - сценарии прав доступа;
  - idempotent reconnect.

## 6) Команды для локальной проверки

- Все тесты:
  - `go test ./...`
- Точечно:
  - `go test ./internal/spp -run Test`
  - `go test ./internal/session -run Test`
  - `go test ./internal/bt -run Test`
  - `go test ./internal/trace -run Test`

Ручной запуск:
- `go run ./cmd/tws_manager --device /dev/rfcomm0`
- Gio: `make run-gio` или `go run -tags gio ./cmd/tws_manager_gio` (Linux: `vulkan-headers`)
- с явным MAC:
  - `go run ./cmd/tws_manager --device /dev/rfcomm0 --addr XX:XX:XX:XX:XX:XX --channel 15`
- c tracing:
  - `go run ./cmd/tws_manager --log captures/session.ndjson --log-raw`

## 7) Быстрый чеклист для агентов перед PR

1. Изменения ограничены Go-частью (`cmd` + `internal`).
2. Не ослаблены safety guardrails (`--unsafe`, scan limits).
3. Добавлены/обновлены тесты рядом с измененным поведением.
4. `go test ./...` проходит.
5. Для новых parser/feature:
   - есть минимум один fixture/юнит-тест;
   - корректный summary в trace/TUI логах.

## 8) Полезные ориентиры по коду

- Wire/frame: `internal/spp/frame.go`
- Каталог команд и parsing: `internal/spp/spp.go`
- Safety: `internal/spp/device_safety.go`
- Session orchestration: `internal/session/session.go`
- RFCOMM lifecycle: `internal/bt/bt.go`
- Trace/export/redaction: `internal/trace/trace.go`
- UI actions: `internal/ui/tui/tui.go`

