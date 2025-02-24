APP_NAME=elevator
BUILD_DIR=bin
SRC_DIR=./cmd

all: build

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(SRC_DIR)

run: build
	@echo "Running $(APP_NAME)..."
	@$(BUILD_DIR)/$(APP_NAME)

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

fmt:
	@echo "Formatting code..."
	@go fmt $(SRC_DIR)/...

lint:
	@echo "Linting code..."
	@golangci-lint run $(SRC_DIR)/...

test:
	@echo "Running tests..."
	@go test $(SRC_DIR)/...

.PHONY: all build run clean fmt lint test
