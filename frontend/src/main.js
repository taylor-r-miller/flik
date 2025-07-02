// Wait for backend to be available with retry logic
async function waitForBackend(maxRetries = 10) {
  const delays = [50, 100, 200, 500, 1000]; // Exponential backoff with max 1 second
  
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    if (window.go && window.go.main && window.go.main.App) {
      console.log(`Backend ready after ${attempt + 1} attempts`);
      return true;
    }
    
    const delay = delays[Math.min(attempt, delays.length - 1)];
    console.log(`Backend not ready, retrying in ${delay}ms (attempt ${attempt + 1}/${maxRetries})`);
    await new Promise(resolve => setTimeout(resolve, delay));
  }
  
  console.error("Backend failed to become available after maximum retries");
  return false;
}

// Initialize the application
async function initializeApp() {
  console.log("Initializing app, waiting for backend...");
  
  const backendReady = await waitForBackend();
  if (backendReady) {
    // Initial status update
    await updateStatus();
    
    // Start periodic updates only after successful backend connection
    if (!intervalId) {
      intervalId = setInterval(updateStatus, 1000);
      console.log("Started periodic status updates");
    }
  } else {
    // Show error state if backend never becomes ready
    const muteStatus = document.getElementById("muteStatus");
    if (muteStatus) {
      muteStatus.textContent = "⚠️ Backend Unavailable";
      muteStatus.className = "mute-status error";
    }
  }
}

// Try multiple initialization methods to handle different timing scenarios
let initialized = false;

// Method 1: Wait for wails:ready event
if (window.runtime && window.runtime.EventsOn) {
  console.log("Setting up wails:ready event listener");
  window.runtime.EventsOn("wails:ready", async () => {
    if (!initialized) {
      console.log("Wails runtime ready event fired");
      initialized = true;
      await initializeApp();
    }
  });
} else {
  console.log("window.runtime not available immediately");
}

// Method 2: Fallback - try initialization after a short delay
setTimeout(async () => {
  if (!initialized) {
    console.log("Fallback initialization after timeout");
    initialized = true;
    await initializeApp();
  }
}, 100);

// Method 3: Alternative runtime check
function checkForRuntime() {
  if (window.runtime && window.runtime.EventsOn && !initialized) {
    console.log("Runtime became available, setting up event listener");
    window.runtime.EventsOn("wails:ready", async () => {
      if (!initialized) {
        console.log("Late wails:ready event fired");
        initialized = true;
        await initializeApp();
      }
    });
  }
}

// Check periodically for runtime availability
const runtimeCheckInterval = setInterval(() => {
  if (window.runtime) {
    checkForRuntime();
    clearInterval(runtimeCheckInterval);
  }
}, 10);

// Clear the check after 1 second to avoid infinite checking
setTimeout(() => {
  clearInterval(runtimeCheckInterval);
}, 1000);

// Update UI status
async function updateStatus() {
  try {
    const status = await window.go.main.App.GetStatus();

    // Update number buffer display
    const numberBuffer = document.getElementById("numberBuffer");
    numberBuffer.textContent = status.numberBuffer || "";

    // Update microphone dot
    const microphoneDot = document.getElementById("microphoneDot");
    if (status.isMuted) {
      microphoneDot.className = "microphone-dot muted";
    } else {
      microphoneDot.className = "microphone-dot active";
    }
  } catch (error) {
    console.error("Error updating status:", error);
    // Show error state for microphone dot
    const microphoneDot = document.getElementById("microphoneDot");
    microphoneDot.className = "microphone-dot";
  }
}

// Track which keys are currently pressed
const pressedKeys = new Set();

// Handle keyboard events - visual feedback on keydown
document.addEventListener("keydown", async (event) => {
  event.preventDefault();

  // Get the key pressed
  let key = event.key;

  // Handle special keys
  if (key === "Escape") {
    key = "Escape";
  } else if (key === "ArrowLeft") {
    key = "h";
  } else if (key === "ArrowRight") {
    key = "l";
  }

  // Only process valid keys
  const validKeys = [
    "0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
    "h", "l", "m", "Escape"
  ];
  if (!validKeys.includes(key)) {
    return;
  }

  // Prevent key repeat - only process if not already pressed
  if (pressedKeys.has(key)) {
    return;
  }
  
  pressedKeys.add(key);

  // Show visual feedback for key buttons
  const keyButton = document.getElementById(`key-${key}`);
  if (keyButton) {
    keyButton.classList.add("pressed");
  }
});

// Handle keyboard events - execute action on keyup
document.addEventListener("keyup", async (event) => {
  event.preventDefault();

  // Get the key released
  let key = event.key;

  // Handle special keys
  if (key === "Escape") {
    key = "Escape";
  } else if (key === "ArrowLeft") {
    key = "h";
  } else if (key === "ArrowRight") {
    key = "l";
  }

  // Only process if this key was pressed
  if (!pressedKeys.has(key)) {
    return;
  }

  pressedKeys.delete(key);

  // Remove visual feedback
  const keyButton = document.getElementById(`key-${key}`);
  if (keyButton) {
    keyButton.classList.remove("pressed");
  }

  // Only process valid keys
  const validKeys = [
    "0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
    "h", "l", "m", "Escape"
  ];
  if (!validKeys.includes(key)) {
    return;
  }

  try {
    // Send key press to backend
    await window.go.main.App.ProcessKeyPress(key);

    // Update UI
    await updateStatus();
  } catch (error) {
    console.error("Error processing key press:", error);
  }
});

// Handle mouse clicks on key buttons
document.addEventListener("click", async (event) => {
  if (event.target.classList.contains("key-button")) {
    const key = event.target.dataset.key;
    
    // Add visual feedback
    event.target.classList.add("pressed");
    
    // Remove feedback after a short delay
    setTimeout(() => {
      event.target.classList.remove("pressed");
    }, 150);
    
    try {
      // Send key press to backend
      await window.go.main.App.ProcessKeyPress(key);
      
      // Update UI
      await updateStatus();
    } catch (error) {
      console.error("Error processing key press:", error);
    }
  }
});

// Global variable for periodic updates
let intervalId = null;

// Handle dock icon clicks by listening for window focus events
window.addEventListener('focus', () => {
  console.log('Window gained focus - possibly from dock icon click');
  if (window.runtime && window.runtime.EventsEmit) {
    window.runtime.EventsEmit('app:activate');
  }
});

// Handle macOS specific app activation events
document.addEventListener('visibilitychange', () => {
  if (!document.hidden) {
    console.log('Document became visible - handling potential dock activation');
    if (window.runtime && window.runtime.EventsEmit) {
      window.runtime.EventsEmit('app:activate');
    }
  }
});
