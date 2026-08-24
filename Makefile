.PHONY: build-server run-server build-agent run-agent loadtest

build-server:
	go build -o cmd/server/server ./cmd/server

run-server: build-server
	go run ./cmd/server

build-agent:
	go build -o cmd/agent/agent ./cmd/agent

run-agent: build-agent
	go run ./cmd/agent

loadtest-num:
	hey -n 2000 -c 50 -m POST -T "application/json" -D loadtest/payload.json http://localhost:8080/updates

loadtest-dur:
	hey -z 60s -c 50 -m POST -T "application/json" -D loadtest/payload.json http://localhost:8080/updates

profile-heap:
	go tool pprof -http=":9090" -seconds=30 http://localhost:8080/debug/pprof/heap

gen-swagger:
	swag init --output ./swagger/