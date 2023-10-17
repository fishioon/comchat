VERSION=`git rev-parse --short HEAD`
BUILD=`date +%FT%T%z`
DOCKER_TAG="dev"

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

build:
	go build ${LDFLAGS} -o ./bin/ccsrv ./server/
proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/*.proto
image:
	docker build -t comchat:${DOCKER_TAG} .

.PHONY: proto
