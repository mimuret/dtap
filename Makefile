all: build

statik/statik.go: assets/flat.avsc
	bash -c "${GOPATH}/bin/statik -src=assets"
build: statik/statik.go
	docker build -t mimuret/dtap:latest .