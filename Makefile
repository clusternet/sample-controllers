# Copyright 2022 The Clusternet Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.EXPORT_ALL_VARIABLES:
BASEIMAGE ?= alpine:3.13.5
GOVERSION ?=1.17.6
REGISTRY ?= ghcr.io

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: vet
vet:
	@go vet cmd

# Verify all changes
.PHONY: verify
verify:
	hack/verify-all.sh

.PHONY: fmt
fmt:
	@find . -type f -name '*.go'| grep -v "/vendor/" | xargs gofmt -w -s

.PHONY: test
test: vet
	go test -race -coverprofile coverage.out -covermode=atomic ./...

# Run golang lint against code
.PHONY: lint
lint: golangci-lint
	@$(GOLANG_LINT) run \
      --timeout 30m \
      --disable-all \
      -E deadcode \
      -E unused \
      -E varcheck \
      -E ineffassign \
      -E goimports \
      -E gofmt \
      -E misspell \
      -E unparam \
      -E unconvert \
      -E govet \
      -E errcheck \
      -E structcheck

# Build Binary
# Example:
#   make sample-controller
EXCLUDE_TARGET=BUILD OWNERS
CMD_TARGET = $(filter-out %$(EXCLUDE_TARGET),$(notdir $(abspath $(wildcard cmd/*/))))
.PHONY: $(CMD_TARGET)
$(CMD_TARGET):
	@hack/make-rules/build.sh $@


.PHONY: images
images:
	@echo "will build docker images"
	@hack/make-rules/images.sh

# find or download golangci-lint
# download golangci-lint if necessary
golangci-lint:
ifeq (, $(shell which golangci-lint))
	@{ \
	set -e ;\
	export GO111MODULE=on; \
	GOLANG_LINT_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOLANG_LINT_TMP_DIR ;\
	go mod init tmp ;\
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.2 ;\
	rm -rf $$GOLANG_LINT_TMP_DIR ;\
	}
GOLANG_LINT=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANG_LINT=$(shell which golangci-lint)
endif
