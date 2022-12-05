all: generate fmt checks test build

fmt:
	go fmt ./...

checks:
	golangci-lint run

test:
	go test ./...

build:
	go build

docker_build:
	docker build -t mimuret/dtapv2:latest .

cover:
	go mod tidy
	go test -coverprofile=cover.out ./...      
	go tool cover -html=cover.out -o cover.html
	go fmt

schemas =  schemas/input_plugins.json schemas/output_plugins.json schemas/filter_plugins.json
plugin_schemas = $(shell find . -name config-schema.json)

generate: pkg/types/dtap_frame.go

pkg/types/dtap_frame.go: proto/dtap_frame.proto
	protoc --proto_path=./proto --go_opt=Mdnstap.proto=github.com/dnstap/golang-dnstap --go_opt=paths=source_relative --go_out=./pkg/types proto/dtap_frame.proto  

schema/input_plugins.json:
	go run misc/merge-schema/main.go pkg/plugin/input /schemas/input_plugins.json > schemas/input_plugins.json.tmp && mv schemas/input_plugins.json.tmp schemas/input_plugins.json
schema/output_plugins.json:
	go run misc/merge-schema/main.go pkg/plugin/output /schemas/output_plugins.json > schemas/output_plugins.json.tmp && mv schemas/output_plugins.json.tmp schemas/output_plugins.json
schema/filter_plugins.json:
	go run misc/merge-schema/main.go pkg/plugin/filter /schemas/filter_plugins.json > schemas/filter_plugins.json.tmp && mv schemas/filter_plugins.json.tmp schemas/filter_plugins.json
schema: $(schemas)
	ajv compile -s schemas/schema.json --inline-refs=true --spec=draft2019 $(addprefix -r ,$(plugin_schemas)) $(addprefix -r ,$(schemas))