.PHONY: build dev clean install test

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
	codesign --remove-signature ./build/bin/Flik.app
	@echo "Re-signing with entitlements..."
	codesign --entitlements entitlements.plist --force --sign - --identifier com.yourcompany.workefficiency --options runtime ./build/bin/Flik.app
	@echo "✓ Signed successfully"

run:
	./build/bin/Flik.app/Contents/MacOS/Flik

copy:
	cp -r ./build/bin/Flik.app /Applications/Flik.app
