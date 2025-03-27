# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gzond all test lint clean devtools help

GOBIN = ./build/bin
GO ?= latest
GORUN = go run

#? gzond: Build gzond
gzond:
	$(GORUN) build/ci.go install ./cmd/gzond
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gzond\" to launch gzond."

#? all: Build all packages and executables
all:
	$(GORUN) build/ci.go install

#? test: Run the tests
test: all
	$(GORUN) build/ci.go test

#? lint: Run certain pre-selected linters
lint: ## Run linters.
	$(GORUN) build/ci.go lint

#? clean: Clean go cache, built executables, and the auto generated folder
clean:
	go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

#? devtools: Install recommended developer tools
devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install github.com/golang/protobuf/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "hypc" 2> /dev/null || echo 'Please install hypc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

#? help: Get more info on make commands.
help: Makefile
	@echo " Choose a command run in go-zond:"
	@sed -n 's/^#?//p' $< | column -t -s ':' |  sort | sed -e 's/^/ /'