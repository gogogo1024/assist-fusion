# Legacy guard: ensure deprecated ticket-svc directory not resurrected
.PHONY: guard-legacy
guard-legacy:
	bash scripts/check_legacy.sh
bootstrap:
	go mod tidy

# Run services (new scaffold)
run-ticket:
	go run ./rpc/ticket/main.go

run-kb:
	go run ./rpc/kb/main.go

run-ai:
	go run ./rpc/ai/main.go

# Build binaries using per-service build.sh (mirrors Kitex scaffold style)
build-ticket:
	bash rpc/ticket/build.sh

build-kb:
	bash rpc/kb/build.sh

build-ai:
	bash rpc/ai/build.sh

build-all: build-ticket build-kb build-ai

# Regenerate Kitex/Thrift code for all services
# Requires kitex installed (go install github.com/cloudwego/kitex/tool/cmd/kitex@latest)
regen: regen-ticket regen-kb regen-ai

regen-ticket:
	kitex -module github.com/gogogo1024/assist-fusion -service ticket-rpc idl/ticket.thrift

regen-kb:
	kitex -module github.com/gogogo1024/assist-fusion -service kb-rpc idl/kb.thrift

regen-ai:
	kitex -module github.com/gogogo1024/assist-fusion -service ai-rpc idl/ai.thrift

test:
	go test ./...

lint:
	golangci-lint run

.PHONY: verify-gen
verify-gen:
	@./script/verify-gen.sh
