.PHONY: all fmt checks test build docker_build cover clean rpm

VERSION ?= $(shell git describe --tags --always --dirty)
ARCH ?= amd64
OS ?= linux

all: generate fmt checks test build

fmt:
	go fmt ./...

checks:
	golangci-lint run

test:
	ginkgo run ./...

build:
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags="-X main.version=$(VERSION)" -o dtap main.go

docker_build:
	docker build -t mimuret/dtapv2:latest .

cover:
	go mod tidy
	go test -coverprofile=cover.out ./...      
	go tool cover -html=cover.out -o cover.html
	go fmt

clean:
	rm -f dtap
	rm -rf ~/rpmbuild/

rpm: clean
		@echo "Building RPM package..."
		rpmdev-setuptree
		tar --exclude='.git' --exclude='.github' -czf ~/rpmbuild/SOURCES/dtap-$(VERSION).tar.gz .
		rpmbuild -ba misc/packaging/rpm/dtap.spec --define "version $(VERSION)"
		@echo "RPM package built: ~/rpmbuild/RPMS/x86_64/dtap-$(VERSION)-1.el7.x86_64.rpm"