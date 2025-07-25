# LinkTracker

<!-- markdownlint-disable -->

## Как запустить

1. Склонируйте репозиторий:

```bash
git clone <repository_url>
```

2. Cоздайте .env файл:

```
TELEGRAM_BOT_TOKEN=<your_key>
```

3. Получите все зависимости:

```bash
go mod download
```

4. Убедитесь, что свободны порты 8080, 8081:

```bash
netstat -tuln | grep -E '(:8080|:8081)\s'
```

Вывод пустой, значит порты свободны.

5. Запустите скраппер

```bash
go run cmd/scrapper/main.go
```

6. Запустите бота

```bash
export $(cat .env)
go run cmd/scrapper/main.go
```

7. Наслаждайтесь

# README

## 🌟 Overview

**go-z0tedd** is a Go-based project designed to manage and process links, tags, and user activities through a modular architecture. It integrates with external services like GitHub and Stack Overflow, provides a bot interface, and includes robust features such as configuration management, state persistence with Redis, metrics collection, and comprehensive testing.

This repository contains a microservices-style application with components for scraping, notification, API handling, and more — all orchestrated via Docker and built with extensibility in mind.

---

## 📦 Features

- **Link Management**: Add, retrieve, and delete links with associated tags and filters.
- **Tag-Based Querying**: Search and filter links using tags and sorting parameters.
- **External API Clients**: Auto-generated clients for GitHub and Stack Overflow APIs.
- **Bot Integration**: Telegram-style bot support for interacting with the system via commands (e.g., `/list`).
- **Configurable Services**: Configurable timeouts, retry policies, access types, and service addresses.
- **State Management**: Uses **Redis** for state persistence.
- **Event Streaming**: Integrated with **Kafka** for asynchronous communication.
- **Monitoring & Observability**:
  - **Prometheus** metrics endpoint
  - **Grafana** dashboard for monitoring
- **Dockerized Environment**: Full `docker-compose` setup including PostgreSQL, Redis, Kafka, Prometheus, and Grafana.
- **Testing Support**: Unit and integration tests using **testcontainers**, mocks, and linter checks.

---

## 🧱 Architecture

The project follows a clean, layered structure:

```
go-z0tedd/
├── api/                   # OpenAPI specs and Protobuf definitions
├── cmd/                   # Main applications (bot, service, etc.)
├── internal/              # Core business logic
│   ├── config/            # Configuration loader and struct
│   ├── api/               # Auto-generated api clients
│   ├── application/       # Business logic
    │   ├── scrapper/      # Scrapper business logic
    │   └── bot/           # Bot business logic
│   ├── domain/            # Domain logic layer
│   └── infrastructure/    #
        ├── http/          # HTTP handlers
        ├── repository/    # Data access layer
        └─ statemanager/   # Bot statemanager implementations
├── pkg/                   # Shared utilities
├── migrations/            # Migrations scripts
├── prometheus/            # Prometheus config and data
├── grafana/               # Grafana config and data
├── redis/                 # Redis configuration and data
├── kafka/                 # Kafka configuration and data
├── docker-compose.yml     # Full environment orchestration
├── .env                   # Environment variables path
├── bot.Dockerfile         # Dockerfile for bot
├── scrapper.Dockerfile    # Dockerfile for scraper
└── Makefile               # Build and dev automation
```

---

## 🛠️ Technologies Used

| Component        | Technology                               |
| ---------------- | ---------------------------------------- |
| Language         | Go (Golang)                              |
| Framework        | Echo (HTTP router)                       |
| Configuration    | Environment variables                    |
| Logging          | Structured logging (slog)                |
| State Management | Redis                                    |
| Messaging        | Kafka                                    |
| Database         | PostgreSQL                               |
| Monitoring       | Prometheus + Grafana                     |
| API Spec         | OpenAPI 3.0 (YAML)                       |
| Mocking          | `mockery` for interface mocks            |
| Containerization | Docker & Docker Compose                  |
| CI/CD            | GitHub Actions (via `.github/workflows`) |

---

## 🚀 Getting Started

### Prerequisites

- [Go](https://golang.org/) 1.20+
- [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/)
- [make](https://www.gnu.org/software/make/)

### Installation & Setup

1. **Clone the repository:**

```bash
git clone https://github.com/central-university-dev/go-z0tedd.git
cd go-z0tedd
```

2. **Start services using Docker Compose:**

```bash
docker-compose up -d
```

This starts:

- PostgreSQL
- Redis
- Kafka
- Prometheus
- Grafana
- Application services (bot, scrapper, API)

3. **Build the project:**

```bash
make build
```

4. **Run the application:**

```bash
make run
```

5. **Access services:**

- **API**: `http://localhost:8080`, `http://localhost:8081`
- **Grafana Dashboard**: `http://localhost:3000` (username: `admin`, password: `admin`)
- **Prometheus**: `http://localhost:9090`

---

## 🔧 Configuration

All configurations are managed via environment variables.

Example snippet:

```env
export DB_ACCESS_TYPE=SQL
export DB_URL=postgresql://postgres_user:postgres_password@localhost:5430/postgres_db
export REDIS_URL=redis://default@localhost:6379/0
export TELEGRAM_BOT_TOKEN=7555555555:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
export SCRAPPER_BASE_URL=http://localhost:8080
export BOT_BASE_URL=http://localhost:8080
export REDIS_PASSWORD=my_redis_password
export CRON="0/10 * * * *"
export POSTGRES_USER=postgres_user
export POSTGRES_PASSWORD=postgres_password
export POSTGRES_DB=postgres_db
export MESSAGE_TRANSPORT_TYPE=kafka # app.message-transport: kafka, http
export KAFKA_SERVERS=localhost:9092
export KAFKA_TOPIC=messages
export BOT_GROUP_ID=bot_consumer
export SCRAPPER_GROUP_ID=scrapper_consumer
export STATE_MANAGER_TYPE=Redis # In-memory
export SCRAPPER_HTTP_ADDRESS=http://localhost:8080
export TIMEOUT=3s
export RATE_LIMIT=15
export BURST=1
export RETRY_COUNT=10
export INITIAL_RETRY_DELAY=1s
```

Environment variables can override any value (e.g., `REDIS_URL=redis://default@localhost:6379/0`).

---

## 📡 Endpoints (API)

Generated from OpenAPI specs in `api/openapi/v1/`.

### Key Endpoints

| Method   | Path     | Summary                     | Authentication (Header)                   | Success Response               | Error Responses   |
| -------- | -------- | --------------------------- | ----------------------------------------- | ------------------------------ | ----------------- |
| `GET`    | `/links` | Get a list of tracked links | `Tg-Chat-Id` (int64)                      | `200 OK` – `ListLinksResponse` | `400 Bad Request` |
| `POST`   | `/links` | Add tracking for a new link | `Tg-Chat-Id` (int64)                      | `200 OK` – `LinkResponse`      | `400 Bad Request` |
| `DELETE` | `/links` | Remove tracking for a link  | `Tg-Chat-Id` (int64), `link` (URL string) | `200 OK` – `LinkResponse`      | `400 Bad Request` |

---

### 🔗 **Scraper Service Endpoints** _(External Clients)_

| Method | Path                             | Summary                 | Parameters      | Success Response                    | Error Responses |
| ------ | -------------------------------- | ----------------------- | --------------- | ----------------------------------- | --------------- |
| `GET`  | `/repos/{owner}/{repo}`          | Get repository details  | `owner`, `repo` | `200 OK` – `Repository`             | `404 Not Found` |
| `GET`  | `/repos/{owner}/{repo}/commits`  | Get repository commits  | `owner`, `repo` | `200 OK` – Array of `Commit`        | `404 Not Found` |
| `GET`  | `/repos/{owner}/{repo}/activity` | Get repository activity | `owner`, `repo` | `200 OK` – Array of `ActivityEvent` | `404 Not Found` |

---

### 📊 **Monitoring & Health Endpoints**

| Method | Path       | Summary                     | Access                             | Notes                                                    |
| ------ | ---------- | --------------------------- | ---------------------------------- | -------------------------------------------------------- |
| `GET`  | `/metrics` | Prometheus metrics endpoint | Public (exposed on separate port)  | Used for monitoring RED metrics (Rate, Errors, Duration) |
| —      | —          | Liveness/Readiness          | Configured in `docker-compose.yml` | Health checks for services                               |

---

### 🧪 **Request/Response Models (Brief)**

- **`ListLinksResponse`**

  ```json
  {
    "links": [
      {
        "url": "string",
        "tags": ["string"],
        "last_activity": "2025-04-05T12:00:00Z"
      }
    ]
  }
  ```

- **`AddLinkRequest`**

  ```json
  { "url": "string", "tags": ["string"] }
  ```

- **`LinkResponse`**

  ```json
  { "url": "string", "status": "added" }
  ```

- **Headers Required**:
  - `Tg-Chat-Id`: Unique Telegram chat identifier (int64)

---

Example request:

```bash
curl "http://localhost:8080/links?tags=golang&sort=asc"
```

---

## 🤖 Bot Commands

The bot supports interactive commands:

- `/start` – Greet the user
- `/list` – List links
- `/track` – Track link
- `/untrack` – Track link
- `/help` – Show available commands

Built with extensible command handlers and integrated with messaging platforms.

---

## 🧪 Testing

Run tests with:

```bash
make test
```

Includes:

- Unit tests
- Integration tests using **Testcontainers**
- Linter checks (`golangci-lint`)

Test coverage includes:

- Repository layer
- HTTP handlers
- Bot logic
- Client retry/fallback behavior

---

## 📊 Monitoring

- **Metrics**: Exposed at `/metrics` for Prometheus scraping.
- **Grafana**: Pre-configured dashboard for service health, request rates, and Redis/Kafka stats.
- **Logging**: Structured logs with trace IDs and levels (DEBUG, INFO, TRACE, etc.).

---

## 🔄 CI/CD Pipeline

Uses GitHub Actions:

- On push/pull request:
  - Run linters
  - Execute unit and integration tests
  - Build binaries
- Deployments can be extended via workflows in `.github/workflows/go.yaml`.

---

## 📄 License

This project is licensed under the GNU License – see the [LICENSE](LICENSE.md) file for details.

---

## 🙋‍♂️ Contact

Maintainer: **z0tedd**  
Email: z0tedd@gmail.com  
Repository: [github.com/central-university-dev/go-z0tedd](https://github.com/central-university-dev/go-z0tedd)
Repository: [github.com/z0tedd/LinkTracker](https://github.com/z0tedd/LinkTracker)

---

## 🎉 Acknowledgments

- Thanks to the T-Bank Backend Academy and open-source community for tools like Echo, Testcontainers, and Prometheus.
- Inspired by clean architecture and DevOps best practices.

---

🚀 Happy coding!
