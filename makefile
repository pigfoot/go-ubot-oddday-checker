VERSION := $(shell git describe --long | sed -r 's/(.*)-([[:digit:]]+)-g[[:xdigit:]]+/\1\.\2/')
TARGETS_NOVENDOR := $(shell glide novendor)
BASE_REPO = github.com/pigfoot/go-ubot-oddday-checker

all: build

build: clean
	CGO_ENABLED=0 go build -o bin/ubot-oddday-checker -ldflags "-w -X $(BASE_REPO)/version.Version=$(VERSION)" -a -tags netgo -installsuffix nocgo $(BASE_REPO)/cmd/ubot-oddday-checker

clean:
	-rm -rf bin

updatedeps:
	@glide update --strip-vcs --update-vendored --strip-vendor

fmt:
	@echo $(TARGETS_NOVENDOR) | xargs go fmt

.PHONY: all build clean updatedeps fmt
