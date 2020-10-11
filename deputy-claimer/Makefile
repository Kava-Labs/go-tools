install:
	go install

generate-mocks:
	go generate -x ./...

test-unit: generate-mocks
	go test ./...

test-integration:
	@# run go vet first to avoid waiting for containers to spin up before finding there's a typo
	go vet ./...
	cd integration_test && run-tests.sh