package permissions

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework CoreServices
#import <ApplicationServices/ApplicationServices.h>
#import <CoreServices/CoreServices.h>

// Check if accessibility permissions are granted
bool checkAccessibilityPermissions() {
    // Check if we have accessibility permissions
    NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @NO};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}

// Request accessibility permissions (will show system dialog)
void requestAccessibilityPermissions() {
    NSDictionary *options = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @YES};
    AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}
*/
import "C"

// CheckAccessibilityPermissions returns true if the app has accessibility permissions
func CheckAccessibilityPermissions() bool {
	return bool(C.checkAccessibilityPermissions())
}

// RequestAccessibilityPermissions requests accessibility permissions from the user
// This will show the system dialog prompting the user to grant permissions
func RequestAccessibilityPermissions() {
	C.requestAccessibilityPermissions()
}