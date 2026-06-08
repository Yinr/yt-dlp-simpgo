# AGENTS.md - utils

Scope: `utils` package only. See `../AGENTS.md` for shared Go style and repo-level commands.

## STRUCTURE

```
execCmd_win.go      - Windows: ExecCmd with HideWindow
execCmd_nowin.go    - Non-Windows: plain ExecCmd wrapper
execCmd_test.go     - Shared tests (cross-platform)
execCmd_win_test.go - Windows-specific SysProcAttr tests
```

## BUILD TAGS

Platform files must appear as pairs:
- `//go:build windows` in `*_win.go`
- `//go:build !windows` in `*_nowin.go`

Both files export the same symbols so callers do not use build tags.

## ExecCmd CONTRACT

`ExecCmd(exePath string, arg ...string) *exec.Cmd`

- Returns an `*exec.Cmd` ready to run. Never returns nil.
- Windows: sets `syscall.SysProcAttr{HideWindow: true}` to suppress console windows.
- Non-Windows: returns `exec.Command(...)` directly with no extra attributes.

## TESTS

Run from this directory with the platform-specific test file included automatically:

```bash
go test -v .
```

- `execCmd_test.go` verifies argument forwarding on all platforms.
- `execCmd_win_test.go` asserts `HideWindow` is true on Windows builds.

## SHARED STYLE

Do not duplicate root conventions here; root `../AGENTS.md` owns imports, naming, error handling, and global commands.
