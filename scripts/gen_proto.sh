#/bin/bash

protoc -I. --go_out=. --go_opt=paths=source_relative internal/message/raft.proto
