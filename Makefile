.PHONY: all test release package

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
	@go test -count=1 \
		./codec/json \
		./util/glob \
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
