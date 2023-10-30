VERSION=`git rev-parse --short HEAD`
BUILD=`date +%FT%T%z`
DOCKER_TAG="dev"

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

build:
	go build ${LDFLAGS} -o ./bin/ccsrv .
proto:
	protoc -I. \
	  --go_out=. --go_opt=paths=source_relative \
	  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	  --js_out=import_style=commonjs:. \
	  --grpc-web_out=import_style=commonjs,mode=grpcwebtext:. \
	  proto/*.proto
image:
	docker build -t fishioon/comchat:${DOCKER_TAG} .

.PHONY: proto
