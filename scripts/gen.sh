protoc -I=$PWD --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=$PWD $PWD/internal/proto/*
protoc -I=$PWD --go_out=$PWD $PWD/internal/proto/*
