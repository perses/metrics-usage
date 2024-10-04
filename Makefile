# Copyright 2024 The Perses Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO                    ?= go
CUE                   ?= cue
GOCI                  ?= golangci-lint
GOFMT                 ?= $(GO)fmt
MDOX                  ?= mdox
GOOS                  ?= $(shell $(GO) env GOOS)
GOARCH                ?= $(shell $(GO) env GOARCH)
GOHOSTOS              ?= $(shell $(GO) env GOHOSTOS)
GOHOSTARCH            ?= $(shell $(GO) env GOHOSTARCH)
COMMIT                := $(shell git rev-parse HEAD)
DATE                  := $(shell date +%Y-%m-%d)
BRANCH                := $(shell git rev-parse --abbrev-ref HEAD)
VERSION               ?= $(shell cat VERSION)
COVER_PROFILE         := coverage.txt
PKG_LDFLAGS           := github.com/prometheus/common/version
LDFLAGS               := -s -w -X ${PKG_LDFLAGS}.Version=${VERSION} -X ${PKG_LDFLAGS}.Revision=${COMMIT} -X ${PKG_LDFLAGS}.BuildDate=${DATE} -X ${PKG_LDFLAGS}.Branch=${BRANCH}
GORELEASER_PARALLEL   ?= 0

export LDFLAGS
export DATE

all: build

.PHONY: checkformat
checkformat:
	@echo ">> checking go code format"
	! $(GOFMT) -d $$(find . -name '*.go' -not -path "./ui/*" -print) | grep '^'
	@echo ">> running check for CUE file format"
	./scripts/cue.sh --checkformat

.PHONY: checkunused
checkunused:
	@echo ">> running check for unused/missing packages in go.mod"
	$(GO) mod tidy
	@git diff --exit-code -- go.sum go.mod

.PHONY: checkstyle
checkstyle:
	@echo ">> checking code style"
	$(GOCI) run --timeout 5m

.PHONY: fmt
fmt:
	@echo ">> format code"
	$(GOFMT) -w -l $$(find . -name '*.go' -not -path "./ui/*" -print)
	./scripts/cue.sh --fmt

.PHONY: test
test: generate
	@echo ">> running all tests"
	$(GO) test -count=1 -v ./...


.PHONY: build
build:
	@echo ">> build binary"
	CGO_ENABLED=0 GOARCH=${GOARCH} GOOS=${GOOS} $(GO) build -ldflags "${LDFLAGS}" -o ./bin/metrics-usage ./

.PHONY: update-go-deps
update-go-deps:
	@echo ">> updating Go dependencies"
	@for m in $$($(GO) list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all); do \
		$(GO) get -d $$m; \
	done
