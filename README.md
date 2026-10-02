# Courier Flow

Сервис управления курьерскими доставками на Go с использованием gRPC, Protocol Buffers и PostgreSQL. Поддерживает создание и получение доставок, управление статусами и потоковую передачу данных.

## Возможности

- создание доставки;
- получение доставки по ID;
- изменение статуса с проверкой допустимых переходов;
- потоковое получение списка доставок;
- приём последовательности координат курьера;
- двунаправленный обмен координатами и подтверждениями;
- хранение данных в PostgreSQL.

## Технологии

- Go;
- gRPC;
- Protocol Buffers;
- PostgreSQL;
- pgx;
- Docker Compose.

## RPC-методы

| Метод | Тип | Назначение |
|---|---|---|
| `CreateDelivery` | Unary | Создание доставки |
| `GetDelivery` | Unary | Получение доставки по ID |
| `UpdateDeliveryStatus` | Unary | Изменение статуса доставки |
| `ListDelivery` | Server streaming | Потоковая выдача списка доставок |
| `ReportLocation` | Client streaming | Приём координат и возврат итогового отчёта |
| `TrackDelivery` | Bidirectional streaming | Обмен координатами и подтверждениями |

## Архитектура

```mermaid
flowchart LR
    Client[gRPC client]
    Service[DeliveryService]
    Repository[DeliveryRepository]
    PostgresRepository[PostgreSQL repository]
    Database[(PostgreSQL)]

    Client --> Service
    Service --> Repository
    Repository --> PostgresRepository
    PostgresRepository --> Database
```

`DeliveryService` отвечает за проверку входных данных, правила изменения статуса и преобразование ошибок в gRPC-коды. Интерфейс `DeliveryRepository` отделяет бизнес-логику от хранения данных. PostgreSQL-реализация выполняет запросы через `pgxpool`.

## Статусы доставки

```text
CREATED → PICKED_UP → IN_TRANSIT → DELIVERED
```

| Значение | Protobuf enum |
|---:|---|
| `0` | `DELIVERY_STATUS_UNSPECIFIED` |
| `1` | `DELIVERY_STATUS_CREATED` |
| `2` | `DELIVERY_STATUS_PICKED_UP` |
| `3` | `DELIVERY_STATUS_IN_TRANSIT` |
| `4` | `DELIVERY_STATUS_DELIVERED` |

Сервис разрешает только последовательные переходы между статусами.

## Структура проекта

```text
courier-flow/
├── api/
│   ├── delivery.proto
│   ├── delivery.pb.go
│   └── delivery_grpc.pb.go
├── cmd/
│   ├── client/
│   │   └── main.go
│   └── server/
│       └── main.go
├── internal/
│   ├── repository/
│   │   └── postgres/
│   │       └── delivery.go
│   └── service/
│       ├── delivery.go
│       └── repository.go
├── migrations/
│   └── 001_init.sql
├── .env.example
├── docker-compose.yaml
├── go.mod
└── go.sum
```

## Требования

- Go 1.26 или новее;
- Docker;
- Docker Compose;
- `protoc` — только при изменении protobuf-контракта.

## Настройка окружения

Скопируйте пример конфигурации:

```bash
cp .env.example .env
```

Укажите параметры локальной базы:

```dotenv
POSTGRES_DB=courier_flow
POSTGRES_USER=courier
POSTGRES_PASSWORD=your_password
POSTGRES_PORT=5432

DATABASE_URL=postgres://courier:your_password@localhost:5432/courier_flow?sslmode=disable
```

Файл `.env` исключён из Git.

## Запуск

Поднимите PostgreSQL:

```bash
docker compose up -d postgres
```

Загрузите переменные окружения в терминал:

```bash
set -a
source .env
set +a
```

Запустите gRPC-сервер:

```bash
go run ./cmd/server
```

Сервер принимает подключения на `localhost:50051`.

В другом терминале запустите клиент:

```bash
go run ./cmd/client
```

Клиент создаёт доставку, проводит её через все статусы и вызывает потоковые RPC-методы.

## Просмотр данных

Откройте PostgreSQL-консоль:

```bash
docker compose exec postgres psql -U courier -d courier_flow
```

Получите список доставок:

```sql
SELECT id, address, status
FROM deliveries
ORDER BY id;
```

Для выхода из `psql` используйте `\q`.

## Управление базой

Остановить контейнеры с сохранением данных:

```bash
docker compose down
```

Удалить контейнеры вместе с PostgreSQL volume:

```bash
docker compose down -v
```

Очистить таблицу и сбросить последовательность ID:

```sql
TRUNCATE TABLE deliveries RESTART IDENTITY;
```

## Генерация protobuf-кода

Установите плагины генератора:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Сгенерируйте Go-код:

```bash
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  api/delivery.proto
```

Файлы `api/delivery.pb.go` и `api/delivery_grpc.pb.go` создаются автоматически и не редактируются вручную.

## Проверка

```bash
gofmt -w cmd internal
go build ./...
go vet ./...
```
