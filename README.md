# Subscription API

REST API сервис для управления и агрегации подписок пользователей.
Реализовано в рамках тестового задания Effective Mobile.

## Архитектура и технологии
* **Язык:** Go 1.26.1
* **Архитектура:** Clean Architecture (Domain -> Repo -> UseCase -> Transport)
* **База данных:** PostgreSQL 15 (pgxpool)
* **Роутинг:** chi
* **Сериализация:** easyjson
* **Логирование:** slog (структурированные JSON-логи)
* **Инфраструктура:** Docker Compose, автоматические миграции (golang-migrate)

## Запуск проекта

1. Создайте файл `.env` на основе примера:
```bash
   cp .env.example .env
```

2. Сгенерируйте код (если вносили изменения) и запустите контейнеры через `Makefile`:
```bash
make gen
make run
```

## Документация

После запуска проекта Swagger UI доступен по адресу:http://localhost:8080/swagger/index.html
