VERSION		:=$(shell cat .version)
YAML_FILES 	:=$(shell find . ! -path "./vendor/*" -type f -regex ".*y*ml" -print)
REG_URI    	?= example/repo
REPO_NAME  	:=$(shell basename $(PWD))
DB_URI     	?= 
MGRT_NAME  	?=
MGRT_DIR   	:= ./sql/migrations/
MGRT_DIRECTION	?=

all: help

.PHONY: init
init: ## Init tools that used in the project
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
.PHONY: version
version: ## Prints the current version
	@echo $(VERSION)

.PHONY: tidy
tidy: ## Updates the go modules and vendors all dependancies 
	go mod tidy

.PHONY: upgrade
upgrade: ## Upgrades all dependancies 
	go get -d -u ./...
	go mod tidy
	go mod vendor

.PHONY: test
test: tidy ## Runs unit tests
	go test -count=1 -race -covermode=atomic -coverprofile=cover.out ./...

.PHONY: lint
lint: lint-go lint-yaml ## Lints the entire project 
	@echo "Completed Go and YAML lints"

.PHONY: lint
lint-go: ## Lints the entire project using go 
	golangci-lint -c .golangci.yaml run

.PHONY: lint-yaml
lint-yaml: ## Runs yamllint on all yaml files (brew install yamllint)
	yamllint -c .yamllint $(YAML_FILES)

.PHONY: vulncheck
vulncheck: ## Checks for soource vulnerabilities
	govulncheck -test ./...

.PHONY: server
server: ## Runs uncpiled version of the server, needs env [DB_URI]
	go run cmd/server/main.go -dburi $(DB_URI)

.PHONY: image
image: ## Builds the server images
	@echo "Building server image..."
	KO_DOCKER_REPO=$(REG_URI)/$(REPO_NAME)-server \
    GOFLAGS="-ldflags=-X=main.version=$(VERSION)" \
    ko build cmd/server/main.go --image-refs .digest --bare --tags $(VERSION),latest

.PHONY: tag
tag: ## Creates release tag 
	git tag -s -m "version bump to $(VERSION)" $(VERSION)
	git push origin $(VERSION)

.PHONY: tagless
tagless: ## Delete the current release tag 
	git tag -d $(VERSION)
	git push --delete origin $(VERSION)

.PHONY: clean
clean: ## Cleans bin and temp directories
	go clean
	rm -fr ./vendor
	rm -fr ./bin

.PHONY: pblint
pblint: ## Lint and format protobuf files
	@echo "Formating protobuf files..."
	@docker run --rm --volume "$(PWD):/workspace" --workdir /workspace bufbuild/buf format
	@echo "Linting protobuf files..."
	@docker run --rm --volume "$(PWD):/workspace" --workdir /workspace bufbuild/buf lint
	@echo "Finished"

.PHONY: pbgen
pbgen: ## Generate protobuf
	docker run --rm --volume "$(PWD):/workspace" --workdir /workspace bufbuild/buf generate

.PHONY: dbgen
dbgen: ## Compile sql to type-safe code
	@sqlc generate

.PHONY: mgrt-prep
mgrt-prep: ## Prepare migration files, needs env [MGRT_NAME="init schema"]
	migrate create -ext sql -dir $(MGRT_DIR) -seq $(MGRT_NAME)

.PHONY: mgrt
mgrt: ## Migrate schema, needs env [DB_URI="db connect uri"] [MGRT_DIRECTION=up|down]
	migrate -database $(DB_URI) -path $(MGRT_DIR) $(MGRT_DIRECTION)

.PHONY: help
help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
