install:
	go install -mod=readonly ./

test-integration:
	@# run go vet first to avoid waiting for containers to spin up before finding there's a typo
	go vet ./...
	cd test/integration && run-tests.sh