module todo/auth-service

go 1.24.0

replace github.com/you/todo/api-contracts => ../api-contracts

replace github.com/you/todo/common => ../common

require github.com/you/todo/api-contracts v0.0.0-00010101000000-000000000000

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	go.uber.org/atomic v1.11.0 // indirect
)

require (
	google.golang.org/genproto/googleapis/api v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/grpc v1.79.1
)

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/redis/go-redis/v9 v9.18.0
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/protobuf v1.36.11
)
