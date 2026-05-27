BINARY_NAME := env-pull
BUILD_DIR   := ./bin
MODULE      := github.com/harsh-sonkar/env-pull

.PHONY: build test clean run

## build: compile the binary into ./bin/
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

## test: run all tests with the race detector enabled
test:
	go test -race ./...

## clean: remove compiled artifacts
clean:
	rm -rf $(BUILD_DIR)

## run: build and execute the binary, forwarding any ARGS
##   Usage: make run ARGS="<subcommand> [flags]"
run: build
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)
