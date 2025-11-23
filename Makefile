APP_NAME=pr-reviewer-service

.PHONY: help dev down build test lint clean db

help:
	@echo "Available commands:"
	@echo "  dev      - start development environment"
	@echo "  down     - stop services"
	@echo "  build    - build containers"
	@echo "  test     - run tests"
	@echo "  lint     - run linter"
	@echo "  clean    - clean up"
	@echo "  db       - connect to database"

dev:
	docker-compose up --build

down:
	docker-compose down

build:
	docker-compose build

test:
	go test ./...

lint:
	golangci-lint run

clean:
	docker-compose down -v
	docker system prune -f

db:
	docker-compose exec postgres psql -U postgres -d pr_reviewer

logs:
	docker-compose logs -f app

restart: down dev

.DEFAULT_GOAL := help