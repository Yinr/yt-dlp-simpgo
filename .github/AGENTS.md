# .github/AGENTS.md

<!-- Generated: 2026-06-08 | Commit: 2cf4ef4 | Branch: main -->

Scope: GitHub Actions and release automation only. Do not document app source internals here.

## TRIGGER

- Release workflow runs on tag push matching `v*`.
- CI runs `go test ./...` on Ubuntu before any build.

## ENVIRONMENT

- Go version: `1.25` (cached via `go.sum`).
- Ubuntu jobs install Fyne build dependencies:
  `gcc libgl1-mesa-dev xorg-dev libgtk-3-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev pkg-config`.
- Windows cross-compilation on Ubuntu uses `gcc-mingw-w64-x86-64` with `CC=x86_64-w64-mingw32-gcc`.

## ACTIVE BUILD MATRIX

| OS     | GOOS    | GOARCH | Suffix | Notes                    |
|--------|---------|--------|--------|--------------------------|
| ubuntu | linux   | amd64  | none   | Native build             |
| ubuntu | windows | amd64  | .exe   | CGO + mingw; GUI ldflags |

- macOS release entries may exist in the workflow as commented examples only; they are not active release artifacts while commented out.

- Ldflags: `-s -w` for all; Windows adds `-H=windowsgui`.
- Version is injected at link time via `-X main.Version=${TAG}`.

## WINDOWS ICON

- Build step runs `rsrc` to generate `rsrc_windows_${GOARCH}.syso` from `res/icon.ico`.
- Syso files are build artifacts and must not be committed.

## RELEASE

1. Build job uploads each matrix artifact with 1-day retention.
2. Release job downloads all artifacts into `dist/`.
3. Generates `checksums.txt` with `sha256sum`.
4. Publishes via `softprops/action-gh-release@v2` with `generate_release_notes: true`.

## ARTIFACTS

- Executables: active matrix outputs only, currently `yt-dlp-simpgo-linux-amd64` and `yt-dlp-simpgo-windows-amd64.exe`
- Checksums: `checksums.txt`
