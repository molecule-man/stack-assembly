.PHONY: vendor

GO_TEST = $(shell command -v gotest || echo "go test")

STACKS = aws cloudformation describe-stacks \
		| jq '.Stacks[] | select((.StackName | startswith("stastest-")) or (.Tags[].Key == "STAS_TEST")) | .StackId' -r

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

testaccall: BUILD_ARGS = -tags acceptance
testaccall: testacc
testaccall: cleanup

testaccmock: GODOG_ARGS = --godog.tags=~@nomock
# testaccmock: BUILD_ARGS = -tags awsmock,acceptance -race -timeout 2s -coverpkg $(shell go list ./... | grep -v mock | paste -sd ',' -) -coverprofile=/tmp/cover.out
testaccmock: BUILD_ARGS = -tags awsmock,acceptance -race -timeout 2s
testaccmock: testacc

testaccnomock: GODOG_ARGS = --godog.tags=nomock
testaccnomock: BUILD_ARGS = -tags acceptance
testaccnomock: export STAS_NO_MOCK := yes
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

cleanup: purgebuckets
cleanup: purgetmpbuckets
cleanup: rmstacks

rmstacks:
	$(STACKS) \
		| xargs -r -l aws cloudformation delete-stack --stack-name

purgebuckets:
	$(STACKS) \
		| xargs -r -l aws cloudformation describe-stack-resources \
			--query "StackResources[?ResourceType=='AWS::S3::Bucket'].PhysicalResourceId" \
			--output text --stack-name \
		| xargs -r -l -I % aws s3 rm s3://% --recursive

purgetmpbuckets:
	aws --profile meadmin s3api list-buckets --query 'Buckets' --output json \
		| jq -r '.[]|select(.Name | startswith("stack-assembly-tmp")) | .Name' \
		| xargs -r -l -I % aws s3 rm s3://% --recursive

lint:
	golangci-lint run
