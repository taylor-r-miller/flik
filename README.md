# Flik
# Work Efficiency Utility for macOS

A lightweight utility application for macOS designed to improve work efficiency through keyboard shortcuts. Built with Go and Wails.

## Features

1. **Display Navigation**: Quickly move focus between multiple displays
   - `h` - Move to the display on the left
   - `l` - Move to the display on the right
   - Vim-style repetition: Type a number before the command (e.g., `2h` moves two displays left)

2. **Microphone Control**: Toggle microphone mute with a single key press
   - `m` - Toggle microphone mute/unmute

3. **Minimal UI**: Clean, centered interface that appears when launched
   - `ESC` - Quit the application

## Installation

### Prerequisites

- Go 1.21 or later
- Node.js 16 or later
- Wails CLI v2.8.0 or later
- macOS 10.15 or later

### Setup

1. Install Wails CLI:
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

2. Clone the repository and navigate to the project directory

3. Install dependencies:
```bash
go mod tidy
```

4. Build the application:
```bash
wails build -platform darwin/amd64
```

## Project Structure

```
workefficiency/
├── main.go              # Main application entry point
├── display_mover.go     # Display Fliking logic
├── audio_manager.go     # Microphone control logic
├── wails.json          # Wails configuration
├── go.mod              # Go module file
├── frontend/
│   └── dist/
│       ├── index.html  # UI HTML
│       └── main.js     # UI JavaScript
└── build/              # Build output directory
```

## Usage

1. Launch the application via your preferred method:
   - Set up a keyboard shortcut (e.g., `Cmd+Shift+E`) using macOS Shortcuts, Raycast, Alfred, or similar
   - Launch from Spotlight or Finder
   - Add to your Dock for quick access

2. When the app launches, the utility window appears immediately

3. Use the keyboard shortcuts:
   - `h` or `←` - Move to left display
   - `l` or `→` - Move to right display
   - `m` - Toggle microphone mute
   - `ESC` - Quit application

### Vim-style Repetition

You can prefix commands with numbers for repetition:
- `2h` - Move two displays to the left
- `3l` - Move three displays to the right

## Configuration

### Setting up a Keyboard Shortcut

You can use various methods to assign a keyboard shortcut to launch the app:

1. **macOS Shortcuts app**: Create a shortcut that opens the Work Efficiency app
2. **Raycast**: Add the app as a quicklink with a hotkey
3. **Alfred**: Create a workflow to launch the app
4. **Karabiner-Elements**: Map a key combination to launch the app

### Permissions

The app requires the following macOS permissions:
- **Accessibility**: To control window focus and mouse movement
- **Microphone**: To control microphone settings

Grant these permissions in System Preferences > Security & Privacy.

## Development

### Running in Development Mode

```bash
wails dev
```

### Building for Production

```bash
# For Intel Macs
wails build -platform darwin/amd64

# For Apple Silicon Macs
wails build -platform darwin/arm64

# Universal binary (works on both)
wails build -platform darwin/universal
```

## How It Works

The app uses a simple workflow:
1. Launch the app with your configured keyboard shortcut
2. The window appears immediately, ready for input
3. Press your desired key command
4. The action executes and the app quits

This design keeps the app lightweight and fast, perfect for quick actions during your workflow.
