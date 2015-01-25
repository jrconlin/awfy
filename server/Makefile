SHELL = /bin/sh
GO = go
HERE = $(shell pwd)
BIN = $(HERE)/bin
GPM = $(HERE)/gpm
DEPS = $(HERE)/.godeps
GOPATH = $(DEPS):$(HERE)

.PHONY: all build clean test

all: build

$(DEPS):
	@echo "Installing Dependencies..."
	GOPATH=$(GOPATH) $(GPM) install

build: $(DEPS)
	GOPATH=$(GOPATH) $(GO) build awfy_server.go
