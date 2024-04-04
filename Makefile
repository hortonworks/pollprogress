export BINARY=pollprogress

# This version refers to the next release version required,
# which will be increased automatically by the dedicated release job
export VERSION=1.1

BUILD_TIME=$(shell date +%FT%T)
LDFLAGS=-ldflags "-X github.com/hortonworks/pollprogress/main.Version=${VERSION} -X github.com/hortonworks/pollprogress/main.BuildTime=${BUILD_TIME}"
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.git/*")
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

deps:
ifeq ($(shell uname),Linux)
ifeq (, $(shell which gh))
	apt-get update
	apt-get -y install software-properties-common
	apt-key adv --keyserver keyserver.ubuntu.com --recv-key 23F3D4EA75716059
	apt-add-repository https://cli.github.com/packages
	apt update
	apt -y install gh
endif
ifeq (, $(shell which aws))
	apt-get update
	apt-get -y install awscli
endif
endif

formatcheck:
	([ -z "$(shell gofmt -d $(GOFILES_NOVENDOR))" ]) || (echo "Source is unformatted"; exit 1)

format:
	gofmt -w ${GOFILES_NOVENDOR}

vet:
	GO111MODULE=on go vet -mod=vendor ./...

test:
	GO111MODULE=on go test -mod=vendor -timeout 30s -race ./...

build: vet formatcheck test build-darwin build-linux

build-darwin:
	GOOS=darwin GO111MODULE=on go build -a -installsuffix cgo ${LDFLAGS} -o build/Darwin/${BINARY} main.go

build-linux:
	GOOS=linux GO111MODULE=on go build -a -installsuffix cgo ${LDFLAGS} -o build/Linux/${BINARY} main.go

build-docker:
	sleep 60 ## wait for docker service on jenkins slave
	@#USER_NS='-u $(shell id -u $(whoami)):$(shell id -g $(whoami))'
	docker run --rm ${USER_NS} -v "${PWD}":/go/src/github.com/hortonworks/pollprogress -w /go/src/github.com/hortonworks/pollprogress -e VERSION=${VERSION} golang:1.17.13 make build

build-docker-local:
	@#USER_NS='-u $(shell id -u $(whoami)):$(shell id -g $(whoami))'
	docker run --rm ${USER_NS} -v "${PWD}":/go/src/github.com/hortonworks/pollprogress -w /go/src/github.com/hortonworks/pollprogress -e VERSION=${VERSION} golang:1.17.13 make build

install: build ## Installs OS specific binary into: /usr/local/bin
	install build/$(shell uname -s)/$(BINARY) /usr/local/bin

release: 
	make build
	./release.sh

release-docker:	@USER_NS='-u$(shell id -u $(whoami)):$(shell id -g $(whoami))'
release-docker:
	sleep 60 ## wait for docker service on jenkins slave
	docker run --rm ${USER_NS} -v "${PWD}":/go/src/github.com/hortonworks/pollprogress -w /go/src/github.com/hortonworks/pollprogress -e VERSION=${VERSION} -e GITHUB_TOKEN=${GITHUB_TOKEN} -e GH_ENTERPRISE_TOKEN=${GH_ENTERPRISE_TOKEN} -e GH_HOST=${GH_HOST} -e AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} -e AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} -e GO111MODULE=on golang:1.17.13 make deps release

gitPush:
	@if ! git diff-index --quiet HEAD Makefile; then\
		git add Makefile;\
		git commit -m "Increase PollProgress version";\
		git push origin HEAD:$(GIT_BRANCH);\
	else \
		echo No changes in Makefile, no git push needed.;\
	fi

mod-tidy:
	GO111MODULE=on go mod tidy -v
	GO111MODULE=on go mod vendor

.PHONY: build
