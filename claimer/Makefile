AWS:=aws
AWS_REGION:=us-east-1
AWS_ACCOUNT_ID=$(shell aws sts get-caller-identity --query 'Account' --output text)
DOCKER:=docker
IMAGE_NAME:=claimer
COMMIT_ID_SHORT:=$(shell git rev-parse --short HEAD)
DOCKER_REPOSITORY_URL:=$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE_NAME)

.PHONY: install
install:
	go install ./

.PHONY: test-integration
test-integration:
	@# run go vet first to avoid waiting for containers to spin up before finding there's a typo
	go vet -tags integration ./...
	cd test/integration && ./run-tests.sh

.PHONY: docker-login
docker-login:
	$(AWS) ecr get-login-password --region $(AWS_REGION) | \
	docker login --username AWS --password-stdin \
	$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com

.PHONY: docker-build
docker-build:
	$(DOCKER) build -t $(IMAGE_NAME):$(COMMIT_ID_SHORT) .

.PHONY: docker-tag
docker-tag:
	$(DOCKER) tag $(IMAGE_NAME):$(COMMIT_ID_SHORT) $(DOCKER_REPOSITORY_URL):$(COMMIT_ID_SHORT)

.PHONY: docker-push
docker-push:
	$(DOCKER) push $(DOCKER_REPOSITORY_URL):$(COMMIT_ID_SHORT)

.PHONY: docker-build-and-push
docker-build-and-push:
	cd ..; $(DOCKER) buildx build --platform linux/amd64  -t $(DOCKER_REPOSITORY_URL):$(COMMIT_ID_SHORT) -f claimer/Dockerfile --push .
