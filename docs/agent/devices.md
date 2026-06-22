# Supported devices and features

Model resolution order: explicit `--model` → identity packet → Bluetooth name/alias/Fast Pair ID → default empty model.

Source of truth: `internal/spp/models.go` (`knownModels`).

| Codename | Product | Features (subset) |
|----------|---------|-------------------|
| EarOne | Nothing ear (1) | anc, eq |
| EarTwo | Ear (2) | anc, eq, dual |
| EarTwos | Nothing Ear (2024) | anc, eq, spatial, dual, advance_eq |
| EarThree | Ear (3) | anc, eq, spatial, dual, advance_eq, bass |
| EarStick | Ear (stick) | eq, advance_eq |
| EarColor | Nothing Ear (a) | anc, eq, spatial, dual, advance_eq |
| Flaffy | Nothing ear (open) | eq, dual |
| Elekid | Nothing Headphone (1) | anc, eq, spatial, dual (stereo battery → case) |
| Forretress | Headphone Pro | anc, eq, spatial, headtrack |
| Crobat / Corsola / Donphan / Espeon | CMF line | varies |
| Girafarig, Gligar, … | codename models | see `models.go` |

UI feature commands map to `featureCommands` in `feature_commands.go`. EQ preset max: 3 default, 7 if model has `advance_eq`.
