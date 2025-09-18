bootstrap:
	go mod tidy

run-ticket:
	go run ./services/ticket-svc/main.go

run-kb:
	go run ./services/kb-svc/main.go

run-ai:
	go run ./services/ai-svc/main.go

test:
	go test ./...

lint:
	golangci-lint run
