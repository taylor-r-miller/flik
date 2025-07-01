# Flik

<p align="center">
  <img src="build/appicon.png" alt="Flik Icon" width="128" height="128">
</p>

A macOS utility for quick display management and audio control via global hotkeys.

## Features

**Global Hotkey**: Press `Ctrl+Space` to instantly show/hide the Flik interface from anywhere on your system.

**Vim-Style Controls**:
- `h` - Move active window to left display
- `l` - Move active window to right display  
- `m` - Toggle microphone mute/unmute
- `Escape` - Quit application

**Number Prefixes**: Use number prefixes for repetition (e.g., `3h` moves left 3 times).

**Quick Operations**: Flik automatically hides after each command, keeping your workflow uninterrupted.

## Usage

1. Launch Flik - the interface appears centered on screen
2. Press `Ctrl+Space` from anywhere to bring up Flik when hidden
3. Use keyboard shortcuts to control displays and audio
4. Click dock icon to show Flik when hidden

## Requirements

- macOS
- Accessibility permissions (for display management)

## Built With

- [Wails v2](https://wails.io/) - Go + Web frontend
- [golang.design/x/hotkey](https://github.com/golang-design/hotkey) - Global hotkey support