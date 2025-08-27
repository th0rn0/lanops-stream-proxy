build: go-install

docker-pull:
	docker compose pull

docker-build:
	docker compose build

go-install:
	docker compose run --rm go mod download
