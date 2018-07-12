GOFILES = $$(go list ./... | grep -v /vendor/)

test:
	go test ${GOFILES}

exec:
	go run cmd/*.go -f Claws.toml -f tpls/cfg.toml

info:
	go run cmd/*.go -f Claws.toml -f tpls/cfg.toml -i

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
