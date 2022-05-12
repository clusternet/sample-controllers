SHELL := /bin/bash
GO := go
GOOS ?= linux
CGO_ENABLED ?= 0

IMAGE := ghcr.io/sample-controller
TAG := v0.1
GITID := `git rev-parse --short HEAD`
PKG_NAME := feedinventory-controller
LDFLAGS := "-extldflags -static -X 'main.BuildTime=`date +"%Y-%m-%d %H:%m:%S"`' -X 'main.Version=$(TAG)-$(GITID)'"

.PHONY: build
build:
	@echo "===========> build bin to ./bin/"
	@echo "===========> build @GOOS=$(GOOS) CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/$(PKG_NAME) -ldflags $(LDFLAGS) ."
	@echo "===========> version is : $(TAG)"
	@GOOS=$(GOOS) CGO_ENABLED=$(CGO_ENABLED) go build -o ./bin/$(PKG_NAME) -ldflags $(LDFLAGS) .

.PHONY: image
image: build
	@echo "===========> build image $(IMAGE):$(TAG)"
	@docker build -f script/Dockerfile -t $(IMAGE):$(TAG) .
	@echo "===========> push image $(IMAGE):$(TAG)"
	

.PHONY: local
local:
	@go build -o ./$(PKG_NAME) -ldflags $(LDFLAGS) .

.PHONY: clean
clean:
	@rm -rf bin/*