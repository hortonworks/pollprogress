VERSION=0.1.2
OWNER=$(shell glu info| sed -n "s/Owner: //p")

deps:
	go get -u github.com/kardianos/govendor
	go get -u github.com/gliderlabs/glu

build:
	glu build $(shell uname)

install: build
	cp build/$(shell uname)/pollprogress /usr/local/bin
	
release:
	glu build linux,darwin
	glu release

generate-license:
	@echo $(shell curl -sH "Accept: application/vnd.github.drax-preview+json" https://api.github.com/licenses/mit | jq .body |sed "s/\[year\] \[fullname\]/$(shell date +%Y) $(OWNER)/" ) > LICENSE

.PHONY: build release
