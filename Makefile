ARCH_AMD = amd64
ARCH_ARM = arm64
BIN  = bin/hermod
BIN_LINUX  = $(BIN)-linux-$(ARCH_AMD)
BIN_DARWIN_AMD = $(BIN)-darwin-$(ARCH_AMD)
BIN_DARWIN_ARM = $(BIN)-darwin-$(ARCH_ARM)
IMAGE = registry.usw.co/uswitch/hermod

SOURCES = $(shell find . -type f -iname "*.go")

.PHONY: clean

all: build

$(BIN_DARWIN_ARM): $(SOURCES)
	GOARCH=$(ARCH_ARM) GOOS=darwin CGO_ENABLED=0 go build -o $(BIN_DARWIN_ARM) *.go

$(BIN_DARWIN_AMD): $(SOURCES)
	GOARCH=$(ARCH_AMD) GOOS=darwin go build -o $(BIN_DARWIN_AMD) *.go

$(BIN_LINUX): $(SOURCES)
	GOARCH=$(ARCH_AMD) GOOS=linux CGO_ENABLED=0 go build -o $(BIN_LINUX) *.go

build: $(BIN_DARWIN_AMD) $(BIN_DARWIN_ARM) $(BIN_LINUX) fmt vet

run:
	go run *.go --kubeconfig ~/.kube/config

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/

image: 
	docker build -t $(IMAGE):testing .
