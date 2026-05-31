DOCKER_COMPOSE = docker compose


.DEFAULT_GOAL := help
.PHONY: help run stop lint tidy gen

help:
	@echo "Доступные команды:"
	@echo "	make run	- запуск проекта и инфраструктуры в Docker"
	@echo "	make stop	- Остановка всех контейнеров и удаление volume-ов"
	@echo "	make lint	- Прогон линтера"
	@echo "	make tidy	- Обновление и загрузка Go-зависимостей"
	@echo "	make gen	- Генерация кода и документации"

run:
	$(DOCKER_COMPOSE) up -d --build

stop:
	$(DOCKER_COMPOSE) down -v

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

gen:
	go generate ./...
	swag init -g cmd/subscription-api/main.go
