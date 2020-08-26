install:
	go install

test: test_integration

test_integration:
	@# check there are no compile errors before initializing the slower integration tests
	go vet -tags integration ./...
	cd integration_test && run-tests.sh

