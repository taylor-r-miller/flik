package audio

import (
	"fmt"
	"os/exec"
	"strings"
)

// Manager handles audio-related functionality
type Manager struct {
	isMuted bool
}

// NewManager creates a new Manager instance
func NewManager() *Manager {
	m := &Manager{}
	// Check initial mute state
	m.checkMuteState()
	return m
}

// ToggleMute toggles the microphone mute state
func (m *Manager) ToggleMute() error {
	// Use AppleScript to toggle microphone mute
	// This works with the system-wide microphone setting
	script := `
	on getMicVolume()
		input volume of (get volume settings)
	end getMicVolume

	on setMicVolume(vol)
		set volume input volume vol
	end setMicVolume

	if getMicVolume() = 0 then
		setMicVolume(100)
		return "unmuted"
	else
		setMicVolume(0)
		return "muted"
	end if
	`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to toggle mute: %w", err)
	}

	result := strings.TrimSpace(string(output))
	m.isMuted = result == "muted"

	return nil
}

// IsMuted returns the current mute state
func (m *Manager) IsMuted() bool {
	return m.isMuted
}

// checkMuteState checks the current microphone mute state
func (m *Manager) checkMuteState() {
	script := `input volume of (get volume settings)`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		// Default to unmuted if we can't check
		m.isMuted = false
		return
	}

	volume := strings.TrimSpace(string(output))
	m.isMuted = volume == "0"
}

// SetInputVolume sets the microphone input volume
func (m *Manager) SetInputVolume(volume int) error {
	if volume < 0 {
		volume = 0
	} else if volume > 100 {
		volume = 100
	}

	script := fmt.Sprintf("set volume input volume %d", volume)
	cmd := exec.Command("osascript", "-e", script)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set input volume: %w", err)
	}

	m.isMuted = volume == 0
	return nil
}

// GetInputVolume returns the current microphone input volume
func (m *Manager) GetInputVolume() (int, error) {
	script := `input volume of (get volume settings)`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get input volume: %w", err)
	}

	var volume int
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &volume)
	if err != nil {
		return 0, fmt.Errorf("failed to parse volume: %w", err)
	}

	return volume, nil
}
