install:
	go install

generate-mocks:
	@# -x prints commands as they're executed
	go generate -x ./...

test-unit:
	# note: mocks need to be regenerated if interfaces have changed
	go test ./...

test-integration:
	@# run go vet first to avoid waiting for containers to spin up before finding there's a typo
	go vet ./...
	cd integration_test && run-tests.sh