# Клиент для наушников Nothing (community)

Сделан **для наших** — владельцев **Nothing** / **CMF**, кому нужно локальное управление без официального приложения: батарея, ANC, EQ, dual-подключение и остальное по SPP-протоколу устройства.

### Скорее всего работает со всеми поддерживаемыми моделями, но абсолютной гарантии нет — буду рад баг-репортам

<h1 align="center">Nothing_helper</h1>
<p align="center"><strong>Компактное приложение для наушников **Nothing / CMF** на Linux и macOS. Заряд, ANC, эквалайзер, поиск наушников и управление кнопкой TALK — с компьютера.</strong></p>
<p align="center"><a href="https://github.com/Kuksenok-i-s/nothing_helper/releases/latest">Скачать последнюю версию</a> · <a href="LICENSE">MIT</a></p>

<p align="center">
  <img src="pics/companion.png" width="350" alt="Nothing_helper — dark theme">
  <img src="pics/companion-light.png" width="350" alt="Nothing_helper — light theme">
</p>

Тёмная и светлая темы. Скриншоты интерфейса с демонстрационными данными.

Системные имена Linux-пакетов и команда `tws_manager` сохранены для совместимости при обновлении.

Независимый проект сообщества, не связанный с Nothing Technology Limited. macOS — экспериментальная поддержка; возможности зависят от модели.

## Возможности

- **Nothing_helper** (Gio, `-tags gio`) — компактное управление зарядом и звуком, выбор устройств, диагностика
- **Walkie Talkie (Ear (3))** — управление кнопкой TALK на кейсе. Сначала запустите Bluetooth-микрофон в звонке или программе записи; приложение проверит поток и включит Super Mic. Само приложение звук не записывает.
- **Поиск наушников** — нажмите силуэт наушника: сигнал запускается после проверки датчика ношения и автоматически останавливается
- **System tray** (`-tags systray`) - статус, батарея, reconnect из трея
- **Автоподключение** - поиск Nothing/CMF и reconnect при обрыве
- **Парсинг протокола** - батарея (L/R/case), статус, identity, firmware, ANC, EQ, spatial, dual, lag
- **Трассировка** - NDJSON-сессия TX/RX, экспорт в JSON (`captures/`)
- **Desktop-уведомления** - connect/disconnect, батарея, low-battery (`--notify`)
- **Безопасность по умолчанию** - GET-команды и ограниченный набор валидированных UI SET; raw scan и не-UI SET требуют `--unsafe`

## Требования

| Компонент | Назначение |
|-----------|------------|
| Go **1.26+** | сборка |
| **BlueZ** (`bluetoothctl`, `rfcomm`) | Bluetooth |
| Privileged helper (`polkit`) или `sudo` | bind/release RFCOMM, chown/chmod |
| `vulkan-headers` (Linux) | только для Gio GUI |
| `libayatana-appindicator` | system tray (Arch/Manjaro: `pacman -S libayatana-appindicator`) |

Наушники должны быть **сопряжены** в системе. Канал RFCOMM по умолчанию - **15** (типично для Nothing Ear).

## Быстрый старт

```bash
make build          # bin/tws_manager — GUI + tray
make run            # compact companion
make run-gio-lite   # GUI without system tray
make run ARGS="--addr AA:BB:CC:DD:EE:FF --channel 15"
make test
```

Сначала выполните сопряжение в настройках Bluetooth системы. На вкладке «Устройства» можно найти и выбрать наушники; подключение и настройка RFCOMM выполняются в фоне.

## Сборка

| Цель | Команда | Результат |
|------|---------|-----------|
| Gio + tray | `make build-gio` | `bin/tws_manager` |
| Gio без tray | `make build-gio-lite` | `bin/tws_manager` |
| Тесты | `make test` | `go test ./...` |

## Флаги CLI

Флаги `cmd/tws_manager`:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--device` | `/dev/rfcomm0` | RFCOMM-устройство (`/dev/rfcomm[0-9]+`) |
| `--addr` | - | MAC Bluetooth; пропускает discovery при bind |
| `--channel` | `15` | RFCOMM-канал при создании `--device` |
| `--model` | - | codename, имя продукта или Fast Pair ID |
| `--log` | auto в `captures/` | путь к NDJSON-трассировке |
| `--log-raw` | `false` | включить raw bytes в лог/экспорт |
| `--capture-dir` | `captures` | каталог JSON-экспорта пакетов |
| `--no-probe` | `false` | не слать identity/battery после connect |
| `--query-every` | `0` (или `60s` при `--notify`) | периодический GET_BATTERY, напр. `30s` |
| `--unsafe` | `false` | разрешить unsafe-операции протокола вне GUI |
| `--auto` | `true` | автопоиск и подключение |
| `--notify` | `true` | desktop-уведомления (нужны `gdbus` или `libnotify` / `notify-send`) |
| `--privilege-helper` | `auto` | backend для privileged операций: `sudo`, `polkit`, `auto`, `none` |
| `--privilege-helper-path` | - | путь к `tws_manager_rfcomm_helper` для `polkit` |

Пример с трассировкой:

```bash
go run -tags gio ./cmd/tws_manager --device /dev/rfcomm0 --log captures/session.ndjson --log-raw
```

## Поддерживаемые устройства

Модель определяется по identity, имени Bluetooth или Fast Pair ID. Можно задать явно: `--model EarThree`.

| Codename | Продукт | Основные фичи |
|----------|---------|---------------|
| EarOne | Nothing ear (1) | anc, eq |
| EarTwo | Ear (2) | anc, eq, dual |
| EarTwos | Nothing Ear (2024) | anc, eq, spatial, dual |
| EarThree | Ear (3) | anc, eq, spatial, dual |
| EarStick | Ear (stick) | eq |
| EarColor | Nothing Ear (a) | anc, eq, spatial, dual |
| Flaffy | Nothing ear (open) | eq, dual |
| Elekid | Nothing Headphone (1) | anc, eq, spatial, dual |
| Forretress | Headphone Pro | anc, eq, spatial, headtrack |
| Crobat | CMF Neckband Pro | anc, eq, spatial |
| Corsola | CMF Buds Pro | anc, eq, spatial |
| Donphan | CMF Buds | eq |
| Espeon | CMF Buds Pro 2 | anc, eq, spatial |
| Girafarig, Gligar, … | codename-модели | см. `internal/spp/models.go` |

Feature-команды в UI: `anc`, `eq`, `spatial`, `lag`, `dual` - с учётом capability конкретной модели.

## Интерфейсы

### Nothing_helper

`make run` открывает новый компактный GUI. «Звук» показывает подтверждённый заряд и ANC; «Все настройки» раскрывает EQ, интенсивность ANC, spatial audio, низкую задержку и dual-подключение с учётом модели. «Устройства» — поиск и подключение, «Диагностика» — техническое состояние и JSON-экспорт. Walkie Talkie доступен на Ear (3) при активном потоке Bluetooth-микрофона. На Linux для проверки нужны PipeWire и `pw-dump`, на macOS используется CoreAudio.

Есть тёмная и светлая темы. С поддержкой трея закрытие окна скрывает его; Open companion возвращает окно, Quit завершает приложение. Без трея закрытие завершает работу. Старый Bubble Tea TUI и прежние экраны Gio удалены.

### System tray

Меню: статус, батарея, refresh, reconnect, disconnect, quit. На GNOME может понадобиться расширение AppIndicator.

## Автозапуск и rootless

- Для desktop-автозапуска используется XDG entry: `packaging/common/tws_manager-autostart.desktop` (`--auto --notify --privilege-helper=polkit`).
- GUI-поток по умолчанию использует `--privilege-helper=auto`: сначала polkit helper, затем sudo fallback с запросом пароля в окне.
- Rootless режим предполагает policy/rules и helper:
  - `packaging/common/org.tws_manager.rfcomm.policy`
  - `packaging/common/90-tws_manager.rules`
  - `cmd/tws_manager_rfcomm_helper`
- **Группа `tws_manager` обязательна** для автозапуска без polkit-пароля: правило polkit разрешает bind/release/chown только участникам группы. Без группы каждый bind через `pkexec` будет спрашивать пароль администратора.
- После `sudo usermod -aG tws_manager $USER` нужен **полный logout/login** (перезапуск только GUI недостаточно).
- Проверка: `groups | grep tws_manager`, затем `pkexec /usr/libexec/tws_manager_rfcomm_helper bind --number 0 --addr <MAC> --channel 15 --owner $(id -u):$(id -g)` — без диалога пароля и с появлением `/dev/rfcomm0`.

## Безопасность

- По умолчанию - **GET + валидированные UI SET** (ANC/EQ/spatial/lag/dual). Подробности: [SECURITY.md](SECURITY.md).
- `--unsafe` - не-UI SET и ограниченный raw scan (`0xC0xx`, delay ≥ 200 ms, max 32 команды).
- Опасные команды каталога (factory reset, debug mode) **заблокированы всегда**, даже с `--unsafe`.
- `polkit` helper - рекомендуемый режим для GUI/autostart; `sudo` остаётся fallback-режимом.
- Raw bytes в логах - только с `--log-raw`.

## Open Source Readiness

- Публичный репозиторий содержит только исходный код проекта и материалы документации.
- Для сверки протокола используйте собственные логи и результаты наблюдений; не включайте в PR чужой исходный код.
- Лицензия проекта: [LICENSE](LICENSE) (MIT).

## Пакетирование

Готовые packaging-артефакты лежат в `packaging/`:

- `packaging/debian` - debian control/rules/install scripts
- `packaging/arch/PKGBUILD` - Arch/Manjaro package recipe
- `packaging/fedora/tws_manager.spec` - Fedora RPM spec
- `packaging/macos` - universal `.app` + DMG (только macOS)
- `packaging/common` - общие desktop/polkit/sysusers файлы

Вспомогательные цели:

```bash
make build-helper
make build-gio-package
make package-deb              # .deb -> dist/tws-manager_<version>-1_<arch>.deb
make package-arch             # Arch pkg -> dist/
make package-rpm
make package-macos            # macOS: dist/Nothing_helper-<version>-universal.dmg
make client-bundle-linux      # portable tarball -> dist/Nothing_helper-<version>-linux-amd64.tar.gz
```

Версия для локальной сборки (опционально):

```bash
./scripts/pkg-version.sh      # tag / APP_VERSION / 0.0.0~dev.<sha>
VERSION=0.2.0 make package-macos
PKG_VERSION=0.2.0 make client-bundle-linux
```

### Post-install (rootless)

- **Debian/Ubuntu (`.deb`)**: `postinst` пытается автоматически создать группу `tws_manager` и добавить пользователя. Если не удалось, выводит ручную команду `usermod`.
- **Arch/Manjaro**: пользователь в группу **не добавляется автоматически** (в отличие от `.deb`). После установки пакета выполните `sudo usermod -aG tws_manager $USER` и перелогиньтесь — иначе autostart будет запрашивать polkit-пароль и не создаст `/dev/rfcomm0` для SPP-сессии.
- **Fedora/RPM**: создаётся группа через `sysusers`, а `post` выводит инструкции по добавлению пользователя в `tws_manager`.

## Структура проекта

```
cmd/tws_manager/          Companion entrypoint
cmd/tws_manager_rfcomm_helper/ privileged helper for polkit
internal/
  app/                    флаги, bootstrap, shutdown
  session/                RFCOMM-сессия, read loop, probe
  spp/                    wire-формат, команды, парсеры, модели
  bt/                     bluetoothctl, rfcomm, discovery
  connect/                autodiscover, bind, reconnect
  notify/                 desktop notifications
  trace/                  NDJSON лог, redaction
  security/               валидация MAC, путей, канала
  ui/companion/                 Gio GUI
  ui/tray/                system tray
  ui/presenter/           общий каталог команд для Companion
```

Карта для агентов и инварианты: [AGENTS.md](AGENTS.md).

## Разработка

```bash
go test ./...                              # все тесты
go test ./internal/spp -run Test           # протокол
go test ./internal/session -run Test         # сессия
gofmt -w cmd internal && make test         # форматирование + тесты
```

После изменений в `cmd/` или `internal/` имеет смысл прогнать полный тестовый набор перед коммитом.

## CI / релизы

GitHub Actions в `.github/workflows/`:

| Workflow | Триггер | Что делает |
|----------|---------|------------|
| `ci.yml` | push, pull_request | Linux vet/build/test-race; macOS vet/build/test; smoke-сборка Debian `.deb` (amd64) |
| `release-client-linux.yml` | tag `v*`, вручную | `.deb` (amd64/arm64), Arch pkg (x86_64), portable tarball; публикация в GitHub Releases |
| `release-client-macos.yml` | tag `v*`, вручную | Universal macOS DMG; публикация в GitHub Releases |

Релиз:

```bash
git tag v0.2.0 && git push origin v0.2.0
```

Оба release-workflow прикрепляют артефакты к одному черновику GitHub Release; публикуйте его после проверки всех сборок (`Nothing_helper v<version>`). Ручной запуск: **Actions → Release Linux client / Release macOS client → Run workflow** (опционально переопределить версию).

## Ограничения

- **Linux** — основная платформа (BlueZ, `/dev/rfcommN`, polkit helper).
- **macOS** — экспериментально (IOBluetooth RFCOMM, Gio GUI).
- Не замена официального приложения: OTA, персонализация жестов и часть фич не реализованы.
- Протокол восстановлен по наблюдаемому трафику и поведению устройств; поведение на непроверенных моделях может отличаться.
