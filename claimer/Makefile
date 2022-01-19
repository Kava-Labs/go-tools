DOCKER:=docker
IMAGE_NAME:=claimer
COMMIT_ID_SHORT:=$(shell git rev-parse --short HEAD)
DOCKER_REPOSITORY_URL:=kava/$(IMAGE_NAME)

.PHONY: install
install:
	go install -mod=readonly ./

.PHONY: test-integration
test-integration:
	@# run go vet first to avoid waiting for containers to spin up before finding there's a typo
	go vet -tags integration ./...
	cd test/integration && ./run-tests.sh

.PHONY: docker-build
docker-build:
	cd .. ; $(DOCKER) build -t $(IMAGE_NAME):$(COMMIT_ID_SHORT) -f claimer/Dockerfile .

.PHONY: docker-tag
docker-tag:
	$(DOCKER) tag $(IMAGE_NAME):$(COMMIT_ID_SHORT) $(DOCKER_REPOSITORY_URL):$(COMMIT_ID_SHORT)

.PHONY: docker-push
docker-push:
	$(DOCKER) push $(DOCKER_REPOSITORY_URL):$(COMMIT_ID_SHORT)
