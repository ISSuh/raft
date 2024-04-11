.PHONY: build vendor test mockgen clear

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BIN_DIR =${ROOT_DIR}/bin
SCRIPTS_DIR =${ROOT_DIR}/scripts
VENDOR_DIR =${ROOT_DIR}/vendor
MOCKS_DIR =${ROOT_DIR}/internal/mocks

vendor:
	go mod tidy -v && go mod vendor -v

build:
	proto
	go build -o ${BIN_DIR}/cluster_example cmd/cluster/main.go
	go build  -o ${BIN_DIR}/node_example cmd/node/main.go

test:
	mockgen
	rm -f cover.out
	go test -coverprofile=cover.out -cover ./...
	go tool cover -html=cover.out

mockgen:
	${SCRIPTS_DIR}/gen_mock.sh

proto:
	${SCRIPTS_DIR}/gen_proto.sh

clean:  
	rm -rf ${BIN_DIR}
	rm -rf ${VENDOR_DIR}
	rm -f $(shell find . -name 'mock_*.go')

