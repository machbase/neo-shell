.PHONY: all test release package regen-mock

targets := $(shell ls main)
uname_s := $(shell uname -s)
uname_m := $(shell uname -m)
nextver := $(shell ./scripts/buildversion.sh)

all:
	@for tg in $(targets) ; do \
		make $$tg; \
	done

cleanpackage:
	@rm -rf packages/*

tmpdir:
	@mkdir -p tmp

test: tmpdir
	@go test -count=1 -cover \
		./codec/json \
		./do \
		./util/glob \
		./util/ini \
		./server/security \
		./server/mqttsvr/mqtt

%:
	@./scripts/build.sh $@ $(nextver)

release:
	@echo "release" $*
	./scripts/package.sh neoshell linux amd64 $(nextver)
	./scripts/package.sh neoshell linux arm64 $(nextver)
	./scripts/package.sh neoshell linux arm $(nextver)
	./scripts/package.sh neoshell darwin arm64 $(nextver)
	./scripts/package.sh neoshell darwin amd64 $(nextver)
	./scripts/package.sh neoshell windows amd64 $(nextver)

## Require https://github.com/matryer/moq
regen-mock:
	moq -out ./util/mock/database.go -pkg mock ../neo-spi Database
	moq -out ./util/mock/server.go -pkg mock   ../neo-spi DatabaseServer
	moq -out ./util/mock/client.go -pkg mock   ../neo-spi DatabaseClient
	moq -out ./util/mock/auth.go -pkg mock     ../neo-spi DatabaseAuth
	moq -out ./util/mock/result.go -pkg mock   ../neo-spi Result
	moq -out ./util/mock/rows.go -pkg mock     ../neo-spi Rows
	moq -out ./util/mock/row.go -pkg mock      ../neo-spi Row