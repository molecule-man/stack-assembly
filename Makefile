GOFILES = $$(go list ./... | grep -v /vendor/)

test: preinstall
	go test ${GOFILES}

preinstall:
	go test -i -v ${GOFILES}

exec:
	go run cmd/main.go -cfg tpls/cfg.toml

lint:
	gometalinter \
	--exclude=vendor \
	--skip=vendor \
	--enable=gosimple \
	--enable=misspell \
	--enable=lll \
	--deadline=120s \
	--cyclo-over=8 \
	--line-length=120 \
	./...
