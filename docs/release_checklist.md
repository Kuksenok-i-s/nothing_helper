# Release Checklist

## Legal and policy

- [ ] `LICENSE` exists and matches intended open-source license.
- [ ] Repository does not contain third-party source mirrors or reference dumps.
- [ ] `README.md` includes unofficial/trademark disclaimer.
- [ ] `SECURITY.md` describes privileged model (`polkit` preferred, `sudo` fallback).

## Packaging consistency

- [ ] Debian package installs helper, polkit files, autostart desktop entry.
- [ ] Debian `postinst` creates `tws_manager` group and prints manual fallback.
- [ ] Arch `PKGBUILD` references `tws_manager.install`.
- [ ] Arch `tws_manager.install` prints `usermod` + relogin instructions.
- [ ] Fedora spec installs same helper/polkit assets and prints group instructions in `%post`.

## Technical hygiene

- [ ] `.gitignore` excludes build outputs (`bin/`, local binaries, `*.zip`).
- [ ] Working tree does not contain accidental binaries/archives.
- [ ] `go test ./...` passes.
- [ ] Packaging scripts pass syntax checks (`sh -n` where applicable).

## Final review

- [ ] `git diff --stat` is coherent and release-focused.
- [ ] Changelog/notes mention rootless flow and post-install user guidance.
