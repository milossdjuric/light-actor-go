#!/bin/bash

# Specify absolute path to the protoc-gen-go plugin
PROTOC_GEN_GO="/home/milossdjuric/go/bin/protoc-gen-go"

# Generate Go code for cluster_messages.proto into the cluster directory
protoc --proto_path=./ \
    --go_out=./ \
    --go_opt=paths=source_relative \
    --plugin=protoc-gen-go=$PROTOC_GEN_GO \
    cluster_messages.proto
