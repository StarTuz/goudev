VERSION := 0.2.1

# Build. GUI on Linux requires CGO (Fyne/OpenGL).
.PHONY: build build-linux build-linux-static test lint clean package package-appimage

build:
	go build -o goudev ./cmd/goudev

test:
	go test -v ./...

lint:
	golangci-lint run --timeout=3m

# Static binary for Linux (no GUI, for servers or minimal installs)
build-linux-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o goudev ./cmd/goudev

# Linux with GUI (default build on Linux)
build-linux:
	GOOS=linux GOARCH=amd64 go build -o goudev ./cmd/goudev

# Build DEB and RPM locally (requires goreleaser)
package:
	@command -v goreleaser >/dev/null 2>&1 || { echo "Install goreleaser: https://goreleaser.com/install/"; exit 1; }
	goreleaser release --clean --snapshot

# Build AppImage locally (requires appimagetool).
package-appimage:
	@mkdir -p dist/appimage/AppDir/usr/bin
	@cp -f goudev dist/appimage/AppDir/usr/bin/ 2>/dev/null || \
	 cp -f dist/goudev_linux_amd64_v1/goudev dist/appimage/AppDir/usr/bin/ 2>/dev/null || \
	 { echo "Binary not found in . or dist/. Run 'make build' first."; exit 1; }
	@cp packaging/appimage/AppRun packaging/appimage/goudev.desktop packaging/appimage/goudev.png dist/appimage/AppDir/
	@chmod +x dist/appimage/AppDir/AppRun
	@mkdir -p dist/appimage/AppDir/usr/share/applications
	@cp packaging/appimage/goudev.desktop dist/appimage/AppDir/usr/share/applications/
	@command -v appimagetool >/dev/null 2>&1 || { echo "Install appimagetool: https://github.com/AppImage/appimagetool/releases"; exit 1; }
	appimagetool dist/appimage/AppDir dist/appimage/goudev-$(VERSION)-x86_64.AppImage
	@echo "AppImage: dist/appimage/goudev-$(VERSION)-x86_64.AppImage"

# Full release cycle (local testing)
release: clean test build package package-appimage

clean:
	rm -f goudev
	rm -rf dist
