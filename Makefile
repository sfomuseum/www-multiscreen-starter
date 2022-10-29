vuln:
	govulncheck ./...

cli:
	go build -mod vendor -o bin/server cmd/server/main.go

debug:
	go run cmd/server/main.go -enable-receiver -access-code-ttl 60
