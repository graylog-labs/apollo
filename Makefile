DIST = dist
BIN = apollo
BIN_LINUX = $(BIN)_linux
BIN_DARWIN = $(BIN)_darwin
BIN_WINDOWS = $(BIN)_windows

MAIN = graylog.com/apollo
GOPATH = $(PWD)
BUILD_OPTS =

build: build-linux build-darwin build-windows

build-linux: dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOPATH=$(GOPATH) go build $(BUILD_OPTS) -o $(DIST)/$(BIN_LINUX)_amd64 $(MAIN)
	GOOS=linux GOARCH=386 CGO_ENABLED=0 GOPATH=$(GOPATH) go build $(BUILD_OPTS) -o $(DIST)/$(BIN_LINUX)_386 $(MAIN)

build-darwin: dist
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 GOPATH=$(GOPATH) go build $(BUILD_OPTS) -o $(DIST)/$(BIN_DARWIN)_amd64 $(MAIN)
	GOOS=darwin GOARCH=386 CGO_ENABLED=0 GOPATH=$(GOPATH) go build $(BUILD_OPTS) -o $(DIST)/$(BIN_DARWIN)_386 $(MAIN)

build-windows: dist
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOPATH=$(GOPATH) go build $(BUILD_OPTS) -o $(DIST)/$(BIN_WINDOWS)_amd64.exe $(MAIN)
	GOOS=windows GOARCH=386 CGO_ENABLED=0 GOPATH=$(GOPATH) go build $(BUILD_OPTS) -o $(DIST)/$(BIN_WINDOWS)_386.exe $(MAIN)

dist:
	mkdir -p $(DIST)

clean:
	rm -rf $(DIST)

.PHONY: dist
