.PHONY: build test vet run mocks loadtest docker-up docker-down

build:
	go build ./cmd/sentinel ./mocks/users ./mocks/payments

test:
	go test ./...

vet:
	go vet ./...

run:
	go run ./cmd/sentinel

mocks:
	go run ./mocks/users &
	go run ./mocks/payments &

loadtest:
	k6 run loadtest/k6.js

docker-up:
	docker compose -f deploy/docker-compose.yml up --build

docker-down:
	docker compose -f deploy/docker-compose.yml down
