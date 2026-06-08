# Embedded Resources

Scope: embedded resources only. This directory contains assets embedded at build time or referenced by build scripts.

Treat filenames as contracts: renaming a file here requires matching changes in code or build scripts.

## Files

| File | Consumer | Purpose |
|------|----------|---------|
| `yt-dlp.conf` | `config.go` (`//go:embed res/yt-dlp.conf`) | Default yt-dlp configuration embedded as `defaultYTDLPConf` |
| `yt-dlp-simpgo.ini` | `config.go` (`//go:embed res/yt-dlp-simpgo.ini`) | Default app configuration embedded as `defaultIniConf` |
| `icon.png` | `config.go` (`//go:embed res/icon.png`) | Window icon data embedded as `iconData` |
| `icon.ico` | `Makefile` / CI (`rsrc_windows_*.syso` generation) | Windows executable icon resource |
| `screenshot.png` | `README.md` | Project screenshot, not embedded |

## Build-time embedding

- `config.go` uses `//go:embed` directives to pull `res/yt-dlp.conf`, `res/yt-dlp-simpgo.ini`, and `res/icon.png` into the binary.
- The `Makefile` and release CI invoke `github.com/akavel/rsrc` with `-ico res/icon.ico` to generate `rsrc_windows_amd64.syso`, which the Go linker embeds as the Windows exe icon.

## Resource types

- **Configuration files**: `yt-dlp.conf`, `yt-dlp-simpgo.ini` — plain text, UTF-8, embedded as strings.
- **Icon assets**: `icon.png` (window icon), `icon.ico` (Windows exe icon) — binary, must not be edited without preserving format.
- **Documentation assets**: `screenshot.png` — not embedded, used by `README.md`.

## Renaming warnings

Do not rename files in this directory without updating the corresponding `//go:embed` paths in `config.go` and the `-ico` path in the `Makefile` (and CI workflows). Mismatched names will break the build.

## Adding new resources

1. Place the file in this directory.
2. Add a `//go:embed` directive in `config.go` if it needs to be embedded.
3. Update this `AGENTS.md` to document the new resource and its consumer.
4. Run `go test -v ./...` to verify the build still works.
