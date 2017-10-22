AWS_REGION=eu-west-1
AWS_PROFILE=meadmin
GOFILES = $$(go list ./... | grep -v /vendor/)

export AWS_REGION
export AWS_PROFILE

test: preinstall
	go test ${GOFILES}

preinstall:
	go test -i -v ${GOFILES}

exec:
	go run cmd/main.go -cfg tpls/cfg.toml
