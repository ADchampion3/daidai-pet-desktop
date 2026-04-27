# Pet Desktop

A lightweight desktop pet application that displays an animated sprite walking across your screen. Built with [Wails](https://wails.io/) (Go + Web frontend).

## Features

- Animated pet sprite that walks and stands on your desktop
- Transparent, always-on-top, frameless window
- Drag the pet to reposition it
- System tray menu for quick settings
- Multi-monitor support with DPI awareness
- Configurable size, step size, and walk interval
- Settings persist across sessions

## Screenshots

![Pet Desktop Screenshot](https://via.placeholder.com/400x200?text=Desktop+Pet+walking+on+screen)

## Tech Stack

**Backend:** Go, Wails v2, Win32 API (user32.dll, shell32.dll, shcore.dll)

**Frontend:** Vanilla JavaScript, Vite, CSS

## Prerequisites

- [Go](https://go.dev/dl/) 1.23+
- [Node.js](https://nodejs.org/) 16+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation/) v2

## Getting Started

### Install Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Run in Development Mode

```bash
wails dev
```

### Build

```bash
wails build
```

The executable will be in `build/bin/`.

## Usage

Once launched, the pet appears on your desktop and starts walking around. Right-click the system tray icon to access settings:

| Menu Option | Description |
|---|---|
| Show / Hide | Toggle pet visibility |
| Enable / Disable Drag | Toggle window click-through |
| Size | Scale the pet (80%, 100%, 120%, 150%) |
| Step | Movement step size (10, 20, 30, 40 px) |
| Interval | Walk interval (100, 150, 300, 600 ms) |
| Display | Choose which monitor the pet walks on |
| Quit | Exit the application |

## Project Structure

```
pet-desktop/
├── main.go              # Application entry point, Wails window setup
├── app.go               # Core app logic: movement, drag, config, events
├── movement.go          # Pet movement engine with bounce behavior
├── display.go           # Multi-monitor layout and DPI scaling
├── display_windows.go   # Win32 monitor enumeration
├── settings.go          # Default config, presets, normalization
├── window_windows.go    # Win32 window click-through and positioning
├── window_other.go      # Stub for non-Windows platforms
├── tray_windows.go      # Win32 system tray with popup menu
├── tray_other.go        # Stub for non-Windows platforms
├── *_test.go            # Unit tests
├── frontend/
│   ├── index.html       # Single-page HTML
│   ├── src/
│   │   ├── main.js      # Sprite rendering and drag handling
│   │   ├── style.css    # Transparent overlay styles
│   │   └── assets/      # Pet sprite images
│   └── vite.config.js   # Vite build config
├── build/               # Wails build assets (icons, manifests, installers)
└── wails.json           # Wails project config
```

## Configuration

Settings are saved to `~/.pet-desktop/config.json`:

```json
{
  "position": { "x": 100, "y": 100 },
  "visible": true,
  "scalePercent": 100,
  "stepSize": 20,
  "walkIntervalMs": 150,
  "dragEnabled": true,
  "displayIndex": 0
}
```

## License

MIT
