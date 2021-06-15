ARCH = amd64
BIN  = bin/hermod
BIN_LINUX  = $(BIN)-linux-$(ARCH)
BIN_DARWIN = $(BIN)-darwin-$(ARCH)
IMAGE = uswitch/hermod

SOURCES = $(shell find . -type f -iname "*.go")

.PHONY: clean

$(BIN_DARWIN): $(SOURCES)
	GOARCH=$(ARCH) GOOS=darwin go build -o $(BIN_DARWIN) main.go

$(BIN_LINUX): $(SOURCES)
	GOARCH=$(ARCH) GOOS=linux CGO_ENABLED=0 go build -o $(BIN_LINUX) main.go

build: $(BIN_DARWIN) $(BIN_LINUX) fmt vet

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/

image: 
	docker build -t registry.usw.co/cloud/hermod:testing .
