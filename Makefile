.PHONY: docker-up docker-down test

docker-up:
	docker compose up --build

docker-down:
	docker compose down

test:
	go test ./... -cover

test-integration:
	go test -tags=integration ./integration -count=1