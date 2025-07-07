.PHONY: build dev clean install test copy install-app uninstall verify-entitlements test-run debug-run sign-minimal build-minimal debug-installed compare-entitlements check-bundle diagnose copy-alt copy-local

# Default target
all: build

# Build the application
build:
	wails build -clean

# Build for specific platforms
build-intel:
	wails build -platform darwin/amd64 -clean

build-arm:
	wails build -platform darwin/arm64 -clean

build-universal:
	wails build -platform darwin/universal -clean
build-signed: build sign

# Run in development mode
dev:
	wails dev

# Clean build artifacts
clean:
	rm -rf build/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	go clean

# Install dependencies
install:
	go mod download
	go mod tidy
	cd frontend && npm install

# Run tests
test:
	go test ./...

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Lint code
lint:
	golangci-lint run

# Generate Wails bindings
generate:
	wails generate module


sign:
	@echo "Cleaning extended attributes..."
	xattr -cr ./build/bin/Flik.app
	@echo "Removing existing signature..."
	codesign --remove-signature ./build/bin/Flik.app 2>/dev/null || true
	@echo "Re-signing with entitlements..."
	codesign --entitlements entitlements.plist --force --sign - --identifier com.yourcompany.workefficiency --options runtime --timestamp=none ./build/bin/Flik.app
	@echo "Verifying signature..."
	codesign --verify --verbose ./build/bin/Flik.app
	@echo "Checking entitlements..."
	codesign -d --entitlements - ./build/bin/Flik.app
	@echo "✓ Signed successfully with proper entitlements"

# Sign with minimal entitlements for testing
sign-minimal:
	@echo "Cleaning extended attributes..."
	xattr -cr ./build/bin/Flik.app
	@echo "Removing existing signature..."
	codesign --remove-signature ./build/bin/Flik.app 2>/dev/null || true
	@echo "Re-signing with minimal entitlements..."
	codesign --entitlements entitlements-minimal.plist --force --sign - --identifier com.yourcompany.workefficiency --options runtime --timestamp=none ./build/bin/Flik.app
	@echo "Verifying signature..."
	codesign --verify --verbose ./build/bin/Flik.app
	@echo "Checking entitlements..."
	codesign -d --entitlements - ./build/bin/Flik.app
	@echo "✓ Signed successfully with minimal entitlements"

# Build with minimal entitlements for troubleshooting
build-minimal: build sign-minimal

# Verify entitlements are properly applied
verify-entitlements:
	@echo "🔍 Verifying entitlements for Flik.app..."
	@if [ ! -d "./build/bin/Flik.app" ]; then \
		echo "❌ Error: Flik.app not found. Run 'make build' first."; \
		exit 1; \
	fi
	@echo "📋 Current entitlements:"
	codesign -d --entitlements - ./build/bin/Flik.app
	@echo "🔐 Signature status:"
	codesign --verify --verbose ./build/bin/Flik.app

# Build, sign and test run the app
test-run: build-signed
	@echo "🧪 Testing signed app..."
	@echo "📊 You can check Console.app for logs (search for 'Flik')"
	./build/bin/Flik.app/Contents/MacOS/Flik

# Debug run with console output
debug-run: build-signed
	@echo "🐛 Running app with debug output..."
	@echo "📝 Logs will appear in terminal and Console.app"
	./build/bin/Flik.app/Contents/MacOS/Flik 2>&1 | tee flik-debug.log

# Debug the installed version in Applications
debug-installed:
	@echo "🐛 Running installed app with debug output..."
	@if [ ! -d "/Applications/Flik.app" ]; then \
		echo "❌ Error: Flik.app not found in Applications. Run 'make copy' first."; \
		exit 1; \
	fi
	@echo "📝 Logs will appear in Console.app (search for 'Flik')"
	/Applications/Flik.app/Contents/MacOS/Flik 2>&1 | tee flik-installed-debug.log

# Compare entitlements between build and installed versions
compare-entitlements:
	@echo "🔍 Comparing entitlements..."
	@if [ ! -d "./build/bin/Flik.app" ]; then \
		echo "❌ Build version not found"; \
		exit 1; \
	fi
	@if [ ! -d "/Applications/Flik.app" ]; then \
		echo "❌ Installed version not found"; \
		exit 1; \
	fi
	@echo "📋 Build version entitlements:"
	codesign -d --entitlements - ./build/bin/Flik.app
	@echo ""
	@echo "📋 Installed version entitlements:"
	codesign -d --entitlements - /Applications/Flik.app
	@echo ""
	@echo "🔐 Build version signature:"
	codesign --verify --verbose ./build/bin/Flik.app
	@echo ""
	@echo "🔐 Installed version signature:"
	codesign --verify --verbose /Applications/Flik.app

# Check bundle structure and dependencies
check-bundle:
	@echo "🔍 Checking bundle structure..."
	@if [ ! -d "./build/bin/Flik.app" ]; then \
		echo "❌ Build version not found"; \
		exit 1; \
	fi
	@echo "📁 Build version bundle structure:"
	find ./build/bin/Flik.app -type f | head -20
	@echo ""
	@echo "🔗 Build version dependencies:"
	otool -L ./build/bin/Flik.app/Contents/MacOS/Flik
	@echo ""
	@if [ -d "/Applications/Flik.app" ]; then \
		echo "📁 Installed version bundle structure:"; \
		find /Applications/Flik.app -type f | head -20; \
		echo ""; \
		echo "🔗 Installed version dependencies:"; \
		otool -L /Applications/Flik.app/Contents/MacOS/Flik; \
	else \
		echo "⚠️  Installed version not found"; \
	fi

# Full diagnosis of the installation issue
diagnose: build-signed copy compare-entitlements check-bundle
	@echo "🩺 Full diagnosis complete!"
	@echo "📝 Check the output above for differences between build and installed versions"
	@echo "🔍 You can also run 'make debug-installed' to see runtime logs"

# Alternative installation method - copy without changing permissions/ownership
copy-alt:
	@echo "🔄 Alternative installation method..."
	@if [ ! -d "./build/bin/Flik.app" ]; then \
		echo "❌ Error: Flik.app not found. Run 'make build' first."; \
		exit 1; \
	fi
	@if [ -d "/Applications/Flik.app" ]; then \
		echo "🗑️  Removing existing Flik.app from Applications..."; \
		sudo rm -rf /Applications/Flik.app; \
	fi
	@echo "📦 Copying Flik.app to Applications (preserving attributes)..."
	sudo cp -pR ./build/bin/Flik.app /Applications/Flik.app
	@if [ -d "/Applications/Flik.app" ]; then \
		echo "✅ Flik successfully installed to Applications!"; \
		echo "🔧 Setting permissions..."; \
		sudo chown -R root:wheel /Applications/Flik.app; \
		sudo chmod -R 755 /Applications/Flik.app; \
	else \
		echo "❌ Installation failed"; \
		exit 1; \
	fi

# Try installation to a different location for testing
copy-local:
	@echo "📍 Installing to ~/Applications for testing..."
	@mkdir -p ~/Applications
	@if [ -d "~/Applications/Flik.app" ]; then \
		rm -rf ~/Applications/Flik.app; \
	fi
	cp -R ./build/bin/Flik.app ~/Applications/Flik.app
	@echo "✅ Flik installed to ~/Applications/Flik.app"
	@echo "🧪 Test with: ~/Applications/Flik.app/Contents/MacOS/Flik"

run:
	./build/bin/Flik.app/Contents/MacOS/Flik

copy:
	@echo "Installing Flik to Applications..."
	@if [ ! -d "./build/bin/Flik.app" ]; then \
		echo "❌ Error: Flik.app not found. Run 'make build' first."; \
		exit 1; \
	fi
	@if [ -d "/Applications/Flik.app" ]; then \
		echo "🗑️  Removing existing Flik.app from Applications..."; \
		sudo rm -rf /Applications/Flik.app; \
	fi
	@echo "📦 Copying Flik.app to Applications..."
	sudo cp -r ./build/bin/Flik.app /Applications/Flik.app
	@if [ -d "/Applications/Flik.app" ]; then \
		echo "✅ Flik successfully installed to Applications!"; \
	else \
		echo "❌ Installation failed"; \
		exit 1; \
	fi

# Build and install in one command
install-app: build-signed copy
	@echo "🎉 Build and installation complete!"

# Uninstall the app from Applications
uninstall:
	@if [ -d "/Applications/Flik.app" ]; then \
		echo "🗑️  Removing Flik from Applications..."; \
		sudo rm -rf /Applications/Flik.app; \
		echo "✅ Flik uninstalled successfully!"; \
	else \
		echo "ℹ️  Flik is not installed in Applications"; \
	fi
