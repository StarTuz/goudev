# Build. GUI on Linux requires CGO (Fyne/OpenGL).
.PHONY: build build-linux build-linux-static clean

build:
	go build -o goudev ./cmd/goudev

# Static binary for Linux (no GUI, for servers or minimal installs)
build-linux-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o goudev ./cmd/goudev

# Linux with GUI (default build on Linux)
build-linux:
	GOOS=linux GOARCH=amd64 go build -o goudev ./cmd/goudev

clean:
	rm -f goudev
