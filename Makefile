UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	LDFLAGS = -extldflags "-static"
endif

GO_BUILDER_VERSION = 1.22.0
NODE_BUILDER_VERSION = 18.19.1-alpine3.19
ALPINE_VERSION = 3.19.1

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: test
test: ## Run go test for all the sub-directories
	@echo "+ $@"
	go test $(TAGS) -v -race -cover -count=1 ./...

.PHONY: clean
clean: ## Clean all artifacts
	@echo "+ $@"
	rm -fr dist

.PHONY: run
run: ## Run git-security
	go run github.com/eekwong/git-security/cmd/git-security

.PHONY: build
build: go ui ## Build git-security

.PHONY: go
go: ## Build go binary only
	@echo "+ $@"
	go build -a \
		-ldflags "$(LDFLAGS)" \
		-o dist/git-security github.com/eekwong/git-security/cmd/git-security

.PHONY: ui
ui: ## Build UI only
	@echo "+ $@"
	$(MAKE) -C cmd/git-security/ui build
	rm -fr dist/ui
	mkdir -p dist/ui
	cp -r cmd/git-security/ui/.output/public/* dist/ui/

.PHONY: image
image: ## Build container image
	@echo "+ $@"
	docker build \
		--no-cache \
		--build-arg GO_BUILDER_VERSION=$(GO_BUILDER_VERSION) \
		--build-arg NODE_BUILDER_VERSION=$(NODE_BUILDER_VERSION) \
		--build-arg ALPINE_VERSION=$(ALPINE_VERSION) \
		-f build/git-security/Dockerfile \
		-t git-security:latest ./

.PHONY: image-amd64
image-amd64: ## Build container image
	@echo "+ $@"
	docker buildx build \
		--platform linux/amd64 \
		--no-cache \
		--build-arg GO_BUILDER_VERSION=$(GO_BUILDER_VERSION) \
		--build-arg NODE_BUILDER_VERSION=$(NODE_BUILDER_VERSION) \
		--build-arg ALPINE_VERSION=$(ALPINE_VERSION) \
		-f build/git-security/Dockerfile \
		-t git-security:latest ./ --load

.PHONY: env-image
env-image: ## Build the demo env image
	@echo "+ $@"
	docker build \
		--no-cache \
		-f build/git-security-env/Dockerfile \
		-t git-security-env:latest build/git-security-env

.PHONY: env-image-amd64
env-image-amd64: ## Build the demo env image
	@echo "+ $@"
	docker buildx build \
		--platform linux/amd64 \
		--no-cache \
		-f build/git-security-env/Dockerfile \
		-t git-security-env:latest build/git-security-env --load
