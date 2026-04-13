.PHONY: help build test lint docker-build docker-push deploy-local deploy-aws terraform-plan terraform-apply clean swagger

REGISTRY ?= videostreamingplatform
VERSION ?= latest
ENVIRONMENT ?= dev
AWS_REGION ?= us-east-1

# Resolve the repository root (parent of this Makefile's directory)
MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
REPO_ROOT := $(abspath $(MAKEFILE_DIR)/..)

help:
	@echo "Video Streaming Platform - Build and Deploy Tasks"
	@echo ""
	@echo "Available targets:"
	@echo "  make build            - Build Go binaries"
	@echo "  make test             - Run tests"
	@echo "  make lint             - Run linters"
	@echo "  make docker-build     - Build Docker images"
	@echo "  make docker-push      - Push Docker images to registry"
	@echo "  make deploy-local     - Deploy to local Kind cluster"
	@echo "  make deploy-aws       - Deploy to AWS EKS"
	@echo "  make terraform-plan   - Plan Terraform changes"
	@echo "  make terraform-apply  - Apply Terraform changes"
	@echo "  make clean            - Clean build artifacts"

build: build-dataservice build-metadataservice
	@echo "✓ All services built"

build-dataservice:
	@echo "Building dataservice..."
	cd $(REPO_ROOT) && CGO_ENABLED=0 GOOS=linux go build -o bin/dataservice ./dataservice

build-metadataservice:
	@echo "Building metadataservice..."
	cd $(REPO_ROOT) && CGO_ENABLED=0 GOOS=linux go build -o bin/metadataservice ./metadataservice

test: test-dataservice test-metadataservice

test-dataservice:
	@echo "Testing dataservice..."
	cd $(REPO_ROOT) && go test -v -race -coverprofile=coverage-dataservice.out ./dataservice/...

test-metadataservice:
	@echo "Testing metadataservice..."
	cd $(REPO_ROOT) && go test -v -race -coverprofile=coverage-metadataservice.out ./metadataservice/...

test-coverage: test
	@echo "Coverage reports generated: coverage-*.out"

lint:
	@echo "Running linters..."
	cd $(REPO_ROOT) && go vet ./...
	cd $(REPO_ROOT) && golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	cd $(REPO_ROOT) && go fmt ./...

swagger:
	@echo "Generating Swagger docs..."
	cd $(REPO_ROOT)/metadataservice && swag init -g main.go -o docs
	cd $(REPO_ROOT)/dataservice && swag init -g main.go -o docs
	@echo "✓ Swagger docs generated"

run: run-dataservice run-metadataservice

run-dataservice:
	@echo "Running dataservice (port 8081 HTTP, 50051 gRPC)..."
	cd $(REPO_ROOT) && go run ./dataservice

run-metadataservice:
	@echo "Running metadataservice (port 8080 HTTP)..."
	cd $(REPO_ROOT) && go run ./metadataservice

docker-build: build-dataservice build-metadataservice
	@echo "Building Docker images..."
	cd $(REPO_ROOT) && docker build -f build/docker/dataservice.Dockerfile -t $(REGISTRY)/dataservice:$(VERSION) .
	cd $(REPO_ROOT) && docker build -f build/docker/metadataservice.Dockerfile -t $(REGISTRY)/metadataservice:$(VERSION) .

docker-push:
	@echo "Pushing Docker images to $(REGISTRY)..."
	docker push $(REGISTRY)/dataservice:$(VERSION)
	docker push $(REGISTRY)/metadataservice:$(VERSION)

deploy-local:
	@echo "Deploying to local Kind cluster..."
	@echo "This requires Kind cluster to be running"
	kubectl create namespace videostreamingplatform --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f $(REPO_ROOT)/k8s/local/
	@echo "Waiting for deployments..."
	kubectl rollout status -f $(REPO_ROOT)/k8s/local/metadata-service-deploy.yaml -n videostreamingplatform
	kubectl rollout status -f $(REPO_ROOT)/k8s/local/data-service-deploy.yaml -n videostreamingplatform

deploy-aws:
	@echo "Deploying to AWS EKS..."
	@echo "Environment: $(ENVIRONMENT)"
	kubectl create namespace videostreamingplatform --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f $(REPO_ROOT)/k8s/aws/manifests/
	@echo "Waiting for deployments..."
	kubectl rollout status deployment/metadata-service -n videostreamingplatform
	kubectl rollout status deployment/data-service -n videostreamingplatform

terraform-plan:
	@echo "Planning Terraform changes for $(ENVIRONMENT)..."
	cd $(REPO_ROOT)/k8s/aws/terraform && \
	terraform init && \
	terraform plan -var-file=terraform.$(ENVIRONMENT).tfvars

terraform-apply:
	@echo "Applying Terraform changes for $(ENVIRONMENT)..."
	cd $(REPO_ROOT)/k8s/aws/terraform && \
	terraform apply -var-file=terraform.$(ENVIRONMENT).tfvars

terraform-destroy:
	@echo "Destroying Terraform resources for $(ENVIRONMENT)..."
	cd $(REPO_ROOT)/k8s/aws/terraform && \
	terraform destroy -var-file=terraform.$(ENVIRONMENT).tfvars

clean: clean-bin clean-coverage clean-docker

clean-bin:
	@echo "Cleaning binary files..."
	rm -rf $(REPO_ROOT)/bin/
	@echo "✓ Cleaned bin/"

clean-coverage:
	@echo "Cleaning coverage files..."
	rm -f $(REPO_ROOT)/coverage*.out
	rm -f coverage-*.out
	@echo "✓ Cleaned coverage files"

clean-docker:
	@echo "Cleaning Docker build artifacts..."
	docker system prune -f
	@echo "✓ Cleaned Docker artifacts"

clean-all: clean
	@echo "Full cleanup completed"
