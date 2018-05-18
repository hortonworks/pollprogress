BINARY=pollprogress
VERSION=0.2.2
OWNER=$(shell glu info| sed -n "s/Owner: //p")

deps:
	go get -u github.com/kardianos/govendor
	go get -u github.com/gliderlabs/glu
	go get -u github.com/sethgrid/multibar

build:
	GOOS=linux CGO_ENABLED=0 go build -o build/Linux/${BINARY} main.go
	GOOS=darwin CGO_ENABLED=0 go build -o build/Darwin/${BINARY} main.go

release: build
	glu release

install: build
	cp build/$(shell uname)/pollprogress /usr/local/bin


generate-license:
	@echo $(shell curl -sH "Accept: application/vnd.github.drax-preview+json" https://api.github.com/licenses/mit | jq .body |sed "s/\[year\] \[fullname\]/$(shell date +%Y) $(OWNER)/" ) > LICENSE

.PHONY: build release
