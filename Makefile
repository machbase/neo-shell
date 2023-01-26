.PHONY: all release

all:
	go build -o ./tmp/neoshell main/*.go

release:
	GOOS=linux GOARCH=amd64 go build -o ./tmp/neoshell_linux_amd64 main/*.go
	GOOS=linux GOARCH=arm64 go build -o ./tmp/neoshell_linux_arm64 main/*.go
	GOOS=darwin GOARCH=amd64 go build -o ./tmp/neoshell_darwin_amd64 main/*.go
	GOOS=darwin GOARCH=arm64 go build -o ./tmp/neoshell_darwin_arm64 main/*.go

