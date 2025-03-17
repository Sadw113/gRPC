# gRPC
Simple gRPC sso service

protoc -I internal/proto/ --go_out=internal/service internal/proto/sso.proto