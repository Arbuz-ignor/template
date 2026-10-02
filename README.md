# TripGo

Лабораторная работа 1. Сервис пока в разработке.

Сейчас реализованы конфигурация из переменных окружения, HTTP-сервер на chi, `/health`, `/ready`, подключение к PostgreSQL через pgxpool и миграции. Типы и серверные интерфейсы сгенерированы из OpenAPI.

Ручки создания, получения и завершения поездки, репозитории и менеджер транзакций пока не реализованы.

## Запуск

Нужны Go 1.26+, Docker, tripgoctl и make. Из корня проекта:

```bash
tripgoctl cluster start
tripgoctl environment start
make migrate
make run
```

В другом терминале проверить сервер:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

`tripgoctl environment start` создаёт `.env` с адресом PostgreSQL. Файл `.env` не добавляется в Git. `make run` читает настройки из `.env.example` и `.env`.

## Конфигурация

Настройки сервера: `HTTP_ADDR`, `HTTP_READ_TIMEOUT`, `HTTP_READ_HEADER_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT`, `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`.

Настройки базы: `DATABASE_URL`, `DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS`, `DATABASE_MAX_CONN_LIFETIME`, `DATABASE_CONNECT_TIMEOUT`, `DATABASE_QUERY_TIMEOUT`.

Пример значений находится в `.env.example`. Настоящее значение `DATABASE_URL` берётся из `.env`, созданного tripgoctl.

## Команды

```bash
make generate
make test
make lint
make migrate
make migrate-down
make run
```

`make generate` обновляет `internal/generated/api.gen.go` по OpenAPI-контракту. `make migrate-down` откатывает одну миграцию за запуск.

## Решения

Уникальный индекс в миграции запрещает одному водителю иметь две активные поездки одновременно.

Выбор уровня изоляции и устройство менеджера транзакций будут описаны после их реализации.