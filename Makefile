.PHONY: vendor

GO111MODULE := on
export GO111MODULE

GO_TEST = $(shell command -v gotest || echo "go test")

build:
	go build $(BUILD_ARGS) -o bin/stas cmd/main.go

test:
	${GO_TEST} ./...

testrace:
	${GO_TEST} -race ./...

run-acctest:
	go test $(BUILD_ARGS) -v ./tests $(GODOG_ARGS)

testacc: clean-testcache
testacc: run-acctest

testaccwip: GODOG_ARGS = --godog.tags=wip --godog.concurrency=1 --godog.format=pretty
testaccwip: BUILD_ARGS = -tags acceptance
testaccwip: testacc
testaccwip: cleanup

testaccmock: GODOG_ARGS = --godog.tags=mock
# testaccmock: BUILD_ARGS = -tags awsmock,acceptance -race -timeout 2s -coverpkg $(shell go list ./... | paste -sd ',' -) -coverprofile=/tmp/cover.out
testaccmock: BUILD_ARGS = -tags awsmock,acceptance -race -timeout 2s
testaccmock: testacc

testaccnomock: GODOG_ARGS = --godog.tags=nomock
testaccnomock: BUILD_ARGS = -tags acceptance
testaccnomock: testacc
testaccnomock: cleanup

testaccshort: GODOG_ARGS = --godog.tags=short
testaccshort: BUILD_ARGS = -tags acceptance
testaccshort: testacc
testaccshort: cleanup

clean-testcache:
	go clean -testcache ./...

test-nocache: clean-testcache
test-nocache: test

cleanup:
	aws cloudformation describe-stacks \
		| jq '.Stacks[] | select(.Tags[].Key == "STAS_TEST") | .StackId' -r \
		| xargs -r -l aws cloudformation delete-stack --stack-name
	aws cloudformation describe-stacks \
		| jq '.Stacks[] | select(.StackName | startswith("stastest-")) | .StackId' -r \
		| xargs -r -l aws cloudformation delete-stack --stack-name

lint:
	golangci-lint run

vendor:
	rm -rf vendor
	go mod vendor
