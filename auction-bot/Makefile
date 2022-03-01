AWS:=aws
AWS_REGION:=us-east-1
AWS_ACCOUNT_ID=$(shell aws sts get-caller-identity --query 'Account' --output text)
DOCKER:=docker
IMAGE_NAME:=auction-bot-service
COMMIT_ID_SHORT:=$(shell git rev-parse --short HEAD)
DOCKER_REPOSITORY_URL:=$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE_NAME)

install:
	go install

.PHONY: docker-login
docker-login:
	$(AWS) ecr get-login-password --region $(AWS_REGION) | \
	docker login --username AWS --password-stdin \
	$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com

.PHONY: docker-build-and-push
docker-build-and-push:
	cd ..; $(DOCKER) buildx build --platform linux/amd64  -t $(DOCKER_REPOSITORY_URL):$(COMMIT_ID_SHORT) -f auction-bot/Dockerfile --push . 
