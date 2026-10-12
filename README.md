# TripGo

Лабораторная работа 1. HTTP API для создания, получения и завершения поездок.

Сервис использует chi, pgxpool и squirrel. Типы и серверный интерфейс генерируются из OpenAPI.

## Запуск

Нужны Go 1.26+, Docker, tripgoctl, make и GCC для тестов с `-race`. В Ubuntu GCC и заголовки устанавливаются командой `sudo apt install build-essential`. Из корня проекта:

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

Проверка поездок:

```bash
curl -i -X POST http://localhost:8080/api/v1/trips -H 'Content-Type: application/json' -d '{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}'
curl -i http://localhost:8080/api/v1/trips/ID
curl -i -X POST http://localhost:8080/api/v1/trips/ID/finish
```

В двух последних командах замените `ID` на `id` из ответа на создание. Повторное создание с тем же `driver_id` должно дать `409 driver_busy`, повторное завершение — `409 trip_completed`.

`tripgoctl environment start` создаёт `.env` с адресом PostgreSQL. Файл `.env` не добавляется в Git. `make run` читает настройки из `.env.example` и `.env`.

## Конфигурация

Настройки сервера: `HTTP_ADDR`, `HTTP_READ_TIMEOUT`, `HTTP_READ_HEADER_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT`, `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`.

Настройки базы: `DATABASE_URL`, `DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS`, `DATABASE_MAX_CONN_LIFETIME`, `DATABASE_CONNECT_TIMEOUT`, `DATABASE_QUERY_TIMEOUT`.

Пример значений находится в `.env.example`. Настоящее значение `DATABASE_URL` берётся из `.env`, созданного tripgoctl.

## Команды

```bash
make generate
make test
make test-db
make lint
make migrate
make migrate-down
make run
```

`make generate` обновляет `internal/generated/api.gen.go` по OpenAPI-контракту. `make migrate-down` откатывает одну миграцию за запуск.
Генерируются только `createTrip`, `getTrip`, `finishTrip`, `health` и `ready`.

`make test` запускает тесты с проверкой гонок. Если `DATABASE_URL` не задан, проверка PostgreSQL пропускается. `make test-db` читает `.env` и проверяет операции, конкурентные запросы и откат транзакций на этой базе; сначала выполните `make migrate`. Проверка создаёт записи со случайными UUID и удаляет их после выполнения.

## Решения

Уникальный индекс в миграции запрещает одному водителю иметь две активные поездки одновременно.

Менеджер транзакций открывает транзакцию с уровнем `READ COMMITTED` и передаёт её репозиторию через `context.Context`. Вложенный вызов использует ту же транзакцию. Ошибка или паника приводят к откату. Запись поездки и первой записи истории статуса происходит в одной транзакции; завершение и запись истории тоже.

`READ COMMITTED` достаточно: частичный уникальный индекс в базе не допускает две активные поездки одного водителя, а `UPDATE ... WHERE status = 'active'` защищает завершение от повторного и конкурентного вызова. При конфликте индекса PostgreSQL возвращает `23505`, который превращается в `driver_busy`. Ошибки API возвращаются с типом `application/problem+json`.

Задания со звёздочкой (идемпотентность по `Idempotency-Key` и Dockerfile) здесь не сделаны.
