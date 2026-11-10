# Trip Service

HTTP-сервис создания, получения и завершения поездок.

## Запуск

Требуются Go 1.26+, GNU Make, компилятор C для тестов с `-race`, Docker и `tripgoctl`. Команды выполняются в Linux или Ubuntu WSL2.

Из корня проекта:

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
```

Оставить `connect` работающим. В другом терминале, из корня проекта:

```bash
make migrate
make run
```

`environment.toml` включает PostgreSQL и observability. `make run` загружает настройки из `.env.example`, затем из `.env`. Адреса БД и коллектора поступают из окружения. `.env` и `.tripgo/` не коммитятся.

## API

В отдельном терминале, из корня проекта:

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
curl -i -X POST http://localhost:8080/api/v1/trips \
  -H 'Content-Type: application/json' -d @examples/trip.json
```

Подставить полученный UUID вместо `ID`:

```bash
curl -i http://localhost:8080/api/v1/trips/ID
curl -i -X POST http://localhost:8080/api/v1/trips/ID/finish
```

`/health` проверяет работу сервера, `/ready` - подключение к БД. Занятый водитель и повторное завершение возвращают `409`, отсутствующая поездка - `404`, неверные данные - `400`.

## Проверки

```bash
make test
make test-integration
make lint
```

`make test` запускает unit-тесты с `-race -count=2` и печатает покрытие домена и usecase. `make test-integration` проверяет репозиторий и транзакции на PostgreSQL из `.env`; нужны работающий `connect` и применённые миграции.

`make lint` при первом запуске устанавливает golangci-lint в `.tools/bin`. `make format` исправляет форматирование. `make generate` обновляет OpenAPI-код и gomock; сгенерированные файлы коммитятся.

## Архитектура

| Пакет | Назначение |
|---|---|
| `internal/domain/trip` | Модель, валидация, переходы статусов и доменные ошибки |
| `internal/usecase/trip/{create,get,finish}` | Бизнес-сценарии |
| `internal/http` | Обработчики, middleware и единый маппинг ошибок в `Problem` |
| `internal/repository/trip` | SQL-запросы через squirrel и pgx |
| `internal/infra` | Пул PostgreSQL, менеджер транзакций, телеметрия и логгер |
| `internal/container` | Создание и связывание зависимостей |

HTTP-обработчики вызывают usecase. Usecase используют домен и интерфейсы зависимостей, не импортируя HTTP или драйвер БД. Репозиторий реализует эти интерфейсы. Домен не зависит от транспорта и инфраструктуры.

Узкие неэкспортируемые интерфейсы объявлены в `contract.go` на стороне потребителя: сценарий определяет нужные методы репозитория, обработчик — методы usecase. Зависимости передаются через конструкторы. `main` читает конфигурацию, вызывает контейнер и управляет запуском и завершением сервера.

Между слоями передаётся `domain/trip.Trip`; отдельный пакет `model` дублировал бы эту структуру. Сгенерированные API-типы используются на границе HTTP. Создание и завершение выполняются в транзакции вместе с записью истории; завершение блокирует поездку через `SELECT FOR UPDATE`.

## Телеметрия

Настройки перечислены в `.env.example`: уровень логов задаёт `LOG_LEVEL`, адрес коллектора — `OTEL_EXPORTER_OTLP_ENDPOINT`, протокол - `OTEL_EXPORTER_OTLP_PROTOCOL`, сэмплирование - `OTEL_TRACES_SAMPLER`.

Middleware выполняются в порядке `Recover → Tracing → Logging → Metrics → router`. Логи пишутся в JSON и содержат `trace_id` и `span_id` внутри спана. HTTP- и SQL-спаны, метрики запросов и поездок экспортируются по OTLP. Недоступность коллектора не мешает обслуживать запросы.

Адреса Jaeger и Grafana печатает `tripgoctl connect`. После создания поездки трейс можно найти в Jaeger по `trace_id` из лога. В Grafana через Prometheus доступны `http_server_request_duration_seconds_count`, `tripgo_trips_created_total` и `tripgo_trips_completed_total`.
