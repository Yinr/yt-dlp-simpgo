
# Build helper scripts

- `build-windows-gui.ps1`: Build a Windows GUI exe that hides the console window. Usage:
  - Open PowerShell in repo root and run `./scripts/build-windows-gui.ps1` (or `./build-windows-gui.ps1`).
- `../build.ps1`: General build helper (supports `-Gui` switch). Examples:
  - `./build.ps1` (normal build)
  - `./build.ps1 -Gui` (windows GUI build, adds `-ldflags "-H=windowsgui"`)

Note: On Windows, double-clicking a GUI exe built with `-H=windowsgui` will not show a console window. Use the `-Gui` option when you want a pure GUI app.
