package display

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics -framework Carbon
#import <CoreGraphics/CoreGraphics.h>
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>

typedef struct {
    double x;
    double y;
    double width;
    double height;
} DisplayBounds;

typedef struct {
    CGDirectDisplayID displayID;
    DisplayBounds bounds;
} DisplayInfo;

// Get current mouse position
void getCurrentMousePosition(double* x, double* y) {
    CGEventRef event = CGEventCreate(NULL);
    CGPoint point = CGEventGetLocation(event);
    CFRelease(event);
    *x = point.x;
    *y = point.y;
}

// Move mouse to position
void moveMouseTo(double x, double y) {
    CGPoint point = CGPointMake(x, y);
    CGEventRef moveEvent = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, point, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, moveEvent);
    CFRelease(moveEvent);
}

// Get all displays
int getDisplays(DisplayInfo* displays, int maxDisplays) {
    CGDirectDisplayID displayList[maxDisplays];
    uint32_t displayCount;

    if (CGGetActiveDisplayList(maxDisplays, displayList, &displayCount) != kCGErrorSuccess) {
        return 0;
    }

    for (int i = 0; i < displayCount && i < maxDisplays; i++) {
        CGRect bounds = CGDisplayBounds(displayList[i]);
        displays[i].displayID = displayList[i];
        displays[i].bounds.x = bounds.origin.x;
        displays[i].bounds.y = bounds.origin.y;
        displays[i].bounds.width = bounds.size.width;
        displays[i].bounds.height = bounds.size.height;
    }

    return displayCount;
}

// Activate window at point
void activateWindowAtPoint(double x, double y) {
    CGPoint point = CGPointMake(x, y);

    CGEventRef clickDown = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDown, point, kCGMouseButtonLeft);
    CGEventRef clickUp = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, point, kCGMouseButtonLeft);

    CGEventPost(kCGHIDEventTap, clickDown);
    usleep(50000); // 50ms delay
    CGEventPost(kCGHIDEventTap, clickUp);

    CFRelease(clickDown);
    CFRelease(clickUp);
}
*/
import "C"
import (
	"fmt"
	"time"
)

// Mover handles moving focus between displays
type Mover struct{}

// NewMover creates a new Mover instance
func NewMover() *Mover {
	return &Mover{}
}

// MoveToDisplay moves focus to the display in the specified direction
func (m *Mover) MoveToDisplay(direction string) error {
	// Get current mouse position
	var currentX, currentY C.double
	C.getCurrentMousePosition(&currentX, &currentY)

	// Get all displays
	maxDisplays := 10
	displays := make([]C.DisplayInfo, maxDisplays)
	displayCount := int(C.getDisplays(&displays[0], C.int(maxDisplays)))

	if displayCount == 0 {
		return fmt.Errorf("no displays found")
	}

	// Find current display
	var currentDisplay *C.DisplayInfo
	for i := 0; i < displayCount; i++ {
		display := &displays[i]
		if float64(currentX) >= float64(display.bounds.x) &&
			float64(currentX) <= float64(display.bounds.x+display.bounds.width) &&
			float64(currentY) >= float64(display.bounds.y) &&
			float64(currentY) <= float64(display.bounds.y+display.bounds.height) {
			currentDisplay = display
			break
		}
	}

	if currentDisplay == nil {
		return fmt.Errorf("could not determine current display")
	}

	// Find target display based on direction
	var targetDisplay *C.DisplayInfo

	if direction == "left" {
		// Find the display to the left
		for i := 0; i < displayCount; i++ {
			display := &displays[i]
			if display.displayID != currentDisplay.displayID {
				// Check if this display is to the left
				if display.bounds.x < currentDisplay.bounds.x {
					// If we haven't found a target yet, or this one is closer
					if targetDisplay == nil || display.bounds.x > targetDisplay.bounds.x {
						targetDisplay = display
					}
				}
			}
		}
	} else if direction == "right" {
		// Find the display to the right
		for i := 0; i < displayCount; i++ {
			display := &displays[i]
			if display.displayID != currentDisplay.displayID {
				// Check if this display is to the right
				if display.bounds.x > currentDisplay.bounds.x {
					// If we haven't found a target yet, or this one is closer
					if targetDisplay == nil || display.bounds.x < targetDisplay.bounds.x {
						targetDisplay = display
					}
				}
			}
		}
	}

	if targetDisplay == nil {
		return fmt.Errorf("no display found in %s direction", direction)
	}

	// Calculate center of target display
	centerX := targetDisplay.bounds.x + targetDisplay.bounds.width/2
	centerY := targetDisplay.bounds.y + targetDisplay.bounds.height/2

	// Move mouse to center of target display
	C.moveMouseTo(centerX, centerY)

	// Small delay to ensure mouse has moved
	time.Sleep(100 * time.Millisecond)

	// Click to activate window at that position
	C.activateWindowAtPoint(centerX, centerY)

	return nil
}

// Display represents a connected display
type Display struct {
	ID     string
	X      string
	Y      string
	Width  string
	Height string
}

// GetConfiguration returns information about connected displays
func (m *Mover) GetConfiguration() ([]Display, error) {
	// Get all displays
	maxDisplays := 10
	cDisplays := make([]C.DisplayInfo, maxDisplays)
	displayCount := int(C.getDisplays(&cDisplays[0], C.int(maxDisplays)))

	if displayCount == 0 {
		return nil, fmt.Errorf("no displays found")
	}

	// Convert to Go structs
	displays := make([]Display, displayCount)
	for i := 0; i < displayCount; i++ {
		displays[i] = Display{
			ID:     fmt.Sprintf("%d", cDisplays[i].displayID),
			X:      fmt.Sprintf("%.0f", cDisplays[i].bounds.x),
			Y:      fmt.Sprintf("%.0f", cDisplays[i].bounds.y),
			Width:  fmt.Sprintf("%.0f", cDisplays[i].bounds.width),
			Height: fmt.Sprintf("%.0f", cDisplays[i].bounds.height),
		}
	}

	return displays, nil
}
