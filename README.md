# Event-Driven Backend System

A production-style event-driven microservices system built with **Go**, **RabbitMQ**, and **PostgreSQL**. Demonstrates fan-out messaging, correlation IDs, idempotency, retry logic with dead-letter queues, and Docker Compose orchestration.

## Architecture

![Architecture Diagram](docs\Diagram.png)

<details>
<summary>📐 PlantUML Source</summary>

@startuml
left to right direction
skinparam componentStyle rectangle

actor Client

component "api-service" as API
database "PostgreSQL" as DB

queue "RabbitMQ Exchange: events" as EX

queue "Queue: crm.user.events" as Q1
queue "Queue: analytics.user.events" as Q2

component "crm-consumer" as CRM
component "analytics-consumer" as AN

database "PostgreSQL" as DB1
database "PostgreSQL" as DB2

queue "DLQ: crm.user.events" as DLQ1
queue "DLQ: analytics.user.events" as DLQ2

Client --> API : HTTP
API --> DB
API --> EX : publish\nuser.created / user.updated

EX --> Q1 : route\nuser.*
EX --> Q2 : route\nuser.*

Q1 --> CRM
Q2 --> AN

CRM --> DB1
AN --> DB2

Q1 --> DLQ1 : fail
Q2 --> DLQ2 : fail
@enduml

</details>

## Features

| Feature                | Implementation                                                                                         |
|------------------------|--------------------------------------------------------------------------------------------------------|
| **Correlation ID**     | Generated/extracted via `X-Correlation-ID` header, passed through RabbitMQ messages, logged everywhere |
| **Idempotency**        | Duplicate event IDs tracked in `idempotency_keys` table and silently ignored                           |
| **Retry + DLQ**        | Failed messages nack'd without requeue → routed to dead-letter queue via RabbitMQ DLX                  |
| **Fan-out**            | Topic exchange routes `user.*` events to both CRM and Analytics queues                                 |
| **Async Processing**   | Consumers process events independently and asynchronously                                              |
| **Decoupled Services** | Each service has its own database, communicates only via events                                        |
| **Simulated Failures** | 10% random failure rate in consumers to demonstrate DLQ behavior                                       |
| **Swagger/OpenAPI**    | API docs at `http://localhost:8080/swagger/index.html`                                                 |

## Services

### api-service (REST API — port 8080)
- `POST /users` — Create a user → publish `user.created`
- `PUT /users/:id` — Update a user → publish `user.updated`
- `GET /users/:id` — Get a user by ID
- `GET /users` — List all users
- `GET /health` — Health check
- `GET /swagger/*` — Swagger UI

### crm-consumer
- Subscribes to `user.created`, `user.updated`, `user.deleted`
- Simulates CRM sync (writes to `crm_sync_log` table)
- Idempotent: deduplicates by `event_id`
- 10% simulated failure rate → messages go to DLQ

### analytics-consumer
- Subscribes to `user.created`, `user.updated`, `user.deleted`
- Aggregates daily metrics (count by event type per day)
- Stores in `analytics_metrics` table
- 10% simulated failure rate → messages go to DLQ

## RabbitMQ Objects

| Object                      | Type           | Purpose                                         |
|-----------------------------|----------------|-------------------------------------------------|
| `events`                    | Topic Exchange | Routes user events by routing key               |
| `crm.user.events`           | Queue          | CRM consumer's main queue                       |
| `analytics.user.events`     | Queue          | Analytics consumer's main queue                 |
| `dlq.crm.user.events`       | Queue (DLQ)    | Dead-letter queue for failed CRM messages       |
| `dlq.analytics.user.events` | Queue (DLQ)    | Dead-letter queue for failed Analytics messages |

**Routing keys:** `user.created`, `user.updated`, `user.deleted`

## How to Run

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)

### Start everything (one command)
```bash
docker compose up --build
```

This starts:
- **PostgreSQL** on port `5432` (databases: `api_db`, `crm_db`, `analytics_db`)
- **RabbitMQ** on port `5672` (management UI: http://localhost:15672 — `guest`/`guest`)
- **api-service** on port `8080`
- **crm-consumer** (background)
- **analytics-consumer** (background)

### Stop everything
```bash
docker compose down -v
```

## Example `curl` Commands

### Create a user
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: my-trace-123" \
  -d '{"email": "john@example.com", "name": "John Doe"}'
```

### Update a user
```bash
curl -X PUT http://localhost:8080/users/<USER_ID> \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: my-trace-456" \
  -d '{"name": "John Updated"}'
```

### Get a user
```bash
curl http://localhost:8080/users/<USER_ID>
```

### List all users
```bash
curl http://localhost:8080/users
```

### Health check
```bash
curl http://localhost:8080/health
```

## Observing the System

1. **Watch logs** — see correlation IDs flow through all services:
   ```bash
   docker compose logs -f
   ```

2. **RabbitMQ Management UI** — http://localhost:15672 (`guest`/`guest`)
   - See exchanges, queues, message rates
   - Inspect DLQ messages

3. **Swagger UI** — http://localhost:8080/swagger/index.html

## Project Structure

```
├── cmd/
│   ├── api-service/          # REST API entry point + Dockerfile
│   ├── crm-consumer/         # CRM consumer entry point + Dockerfile
│   └── analytics-consumer/   # Analytics consumer entry point + Dockerfile
├── internal/
│   ├── api/                  # HTTP handlers and router
│   ├── crm/                  # CRM consumer logic
│   └── analytics/            # Analytics consumer logic
├── pkg/
│   ├── config/               # Environment-based configuration
│   ├── middleware/            # Correlation ID middleware
│   ├── models/               # Shared domain models and events
│   ├── postgres/             # Database connection and migrations
│   └── rabbitmq/             # RabbitMQ connection, publisher, consumer
├── docs/                     # Swagger documentation
├── scripts/                  # Database init scripts
├── docker-compose.yml        # One command to run everything
└── README.md
```
