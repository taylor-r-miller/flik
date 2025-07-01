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

## Installation

### Prerequisites
- Go 1.24+ 
- Node.js (for frontend dependencies)
- macOS (required for display management features)

### Steps
1. **Install Go** (if not already installed)
   ```bash
   # Using Homebrew
   brew install go
   
   # Or download from https://golang.org/dl/
   ```

2. **Install Node.js** (if not already installed)
   ```bash
   # Using Homebrew
   brew install node
   
   # Or download from https://nodejs.org/
   ```

3. **Install Wails CLI**
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

4. **Clone the repository**
   ```bash
   git clone https://github.com/taylor-r-miller/Flik.git
   cd Flik
   ```

5. **Install dependencies**
   ```bash
   go mod tidy
   ```

6. **Build the application**
   ```bash
   # Development build
   wails dev
   
   # Production build
   wails build
   ```

7. **Grant accessibility permissions**
   - Go to System Preferences → Security & Privacy → Privacy → Accessibility
   - Add Flik to the list of applications with accessibility access
   - This is required for display management features

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