GO ?= go
GOFMT ?= gofmt "-s"
TEST_TIMEOUT=-timeout 5m
GOFILES := $(shell find . -name "*.go")
PACKAGES := $(shell grep -Ri "module " * | cut -f2 -d' ')
GODIRS := $(shell find . -name '*.go' | xargs dirname | sort -u)

build:
	sam build

.PHONY: deploy
deploy:
	@echo "Redeploy to AWS"
	@sam deploy --stack-name squyre --capabilities "CAPABILITY_NAMED_IAM"

.PHONY: deploy-guided
deploy-guided:
	@echo "First time deploy to AWS"
	@sam deploy --stack-name squyre --capabilities "CAPABILITY_NAMED_IAM" --guided

test:
	@echo "go test all packages"
	@for DIR in $(GODIRS); do cd $$DIR; go test ${TEST_TIMEOUT} -cover -v -count=1; cd - > /dev/null ; done;

setup:
	@echo "run Squyre setup"
	@cd scripts/bootstrap; go run main.go; cd -

dep-scan:
	@echo "Scan OSV for known security vulnerabilities in our dependencies"
	@go install github.com/google/osv.dev/tools/osv-scanner/cmd/osv-scanner@latest
	@for DIR in $(GODIRS); do ${HOME}/go/bin/osv-scanner --json --lockfile=$$DIR/go.mod ; done;

.PHONY: lint
lint:
	@echo "go lint all packages"
	@hash golint > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) install golang.org/x/lint/golint@latest; \
	fi
	@for DIR in $(GODIRS); do echo $$DIR; `go list -f {{.Target}} golang.org/x/lint/golint` -set_exit_status $$DIR || exit 1; done;
	

.PHONY: fmt
fmt:
	$(GOFMT) -w $(GOFILES)

.PHONY: fmt-check
fmt-check:
	@diff=$$($(GOFMT) -d $(GOFILES)); \
	if [ -n "$$diff" ]; then \
		echo "Please run 'make fmt' and commit the result:"; \
		echo "$${diff}"; \
		exit 1; \
	fi;

vet:
	$(GO) vet $(VETPACKAGES)

################
# Dependencies #
################

get-deps-verify:
	@echo "go get verification utilities"
	go get golang.org/x/lint/golint