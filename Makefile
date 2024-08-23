# Define the base directory (default is current directory)
BASEDIR := $(shell pwd)

# Find all directories containing a main.go file
MODULES := $(shell find . -type f -name 'main.go' -not -path "*/vendor/*" -exec dirname {} \;)

# output the list of modules
.PHONY: modules
modules:
	@echo $(MODULES)

# Default target: build all modules
.PHONY: build-all
build-all:
	@for dir in $(MODULES); do \
    	echo "Building module in $$dir..."; \
    	(cd $$dir && go build -o $(BASEDIR)/$$(basename $$dir)/$$(basename $$dir)) || { echo "Skipping $$dir due to build failure or permission issues."; continue; }; \
    	echo "Built module $$dir successfully."; \
    done

.PHONY: golangci-lint
golangci-lint:
	golangci-lint run

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v ./...
