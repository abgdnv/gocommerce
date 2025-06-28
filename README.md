# gocommerce
[![Go Report Card](https://goreportcard.com/badge/github.com/abgdnv/gocommerce)](https://goreportcard.com/report/github.com/abgdnv/gocommerce)
![CI](https://github.com/abgdnv/gocommerce/actions/workflows/go.yml/badge.svg)
[![License](https://img.shields.io/badge/license-MIT-blue)](https://github.com/abgdnv/gocommerce/blob/main/LICENSE)


A simple yet robust microservices project for an e-commerce platform, built with Go. This project serves as a template for building production-ready microservices, demonstrating best practices in project structure, configuration management, database interaction, and testing.

This repository will host multiple services, such as `product`, `order`, `notification`, etc.

## Features

-   **RESTful API**: Clean and simple APIs for CRUD operations (e.g., on products).
-   **Structured Logging**: Uses `slog` for structured, context-aware logging.
-   **Configuration Management**: Flexible configuration loading from files (`config.yaml`), environment variables, and `.env` files using `koanf`.
-   **PostgreSQL Integration**: Uses `pgx` for high-performance PostgreSQL interaction and `sqlc` to generate type-safe Go code from SQL queries.
-   **Robust Routing**: Employs `chi` for lightweight and powerful HTTP routing and middleware.
-   **Graceful Shutdown**: Implements graceful server shutdown to handle in-flight requests.
-   **Optimistic Locking**: The product service uses a version field for optimistic concurrency control on product updates.
-   **Dockerized Environment**: Includes a `docker-compose.yml` for easy setup of a PostgreSQL database.
-   **Testing**: Comprehensive unit tests with mocks for service and handler layers.
-   **pprof Server**: Optional `pprof` server for profiling and debugging, configurable via `config.yaml` or environment variables (disabled by default).
-   **Linting**: Integrated with `golangci-lint` and `pre-commit` to ensure code quality.

## Prerequisites

-   [Go](https://golang.org/dl/) (version 1.24 or newer)
-   [Docker](https://www.docker.com/get-started) and [Docker Compose](https://docs.docker.com/compose/install/)
-   [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (for regenerating database code)
-   [golang-migrate](https://github.com/golang-migrate/migrate) (for running database migrations)

## Getting Started

The following instructions are for the `product_service`. The primary way to run the services is with Docker Compose.

### 1. Clone the repository

```sh
git clone https://github.com/abgdnv/gocommerce.git
cd gocommerce
```

### 2. Configure the application

Create a `.env` file from the example. This file holds the configuration for Docker Compose and the services.

```sh
cp example.env .env
```

You can modify `.env` if you need to change ports or credentials.

### 3. Run with Docker Compose

Build and start the services in detached mode:

```sh
docker-compose up --build -d
```

This command starts both the `product_service` and its PostgreSQL database.

### 4. Run Database Migrations

With the database container running, apply the migrations. You'll need `golang-migrate` installed locally.

```sh
migrate -path internal/product/migrations -database "postgres://postgres:postgres@localhost:5432/gocommerce?sslmode=disable" up
```

The connection string should match the values in your `.env` file. The service should now be running and accessible.

---

### Development

For local development, you might want to run the Go application directly on your host.

1.  **Start only the database:**
    ```sh
    docker-compose up -d db
    ```
2.  **Run migrations:** (same command as above)
3.  **Run the application:**
    ```sh
    go run ./cmd/product_service/
    ```

### Configuration

The application is configured using `koanf` with the following precedence:
1. Environment variables (e.g., `PRODUCT_SVC_SERVER_PORT`)
2. `.env` file
3. `config.yaml` file

An example `config.yaml` might look like this:

```yaml
# config.yaml
server:
  port: 8080
  maxHeaderBytes: 1048576 # 1MB
  timeout:
    read: 5s
    write: 10s
    idle: 120s
    readHeader: 5s
database:
  url: "postgresql://postgres:postgres@localhost:5432/gocommerce?sslmode=disable"
log:
  level: "info"
pprof:
  enabled: false
  addr: "localhost:6060"
```

#### Configuration Parameters

The following parameters can be set in `config.yaml` or via environment variables (which take precedence). The application does not have built-in default values, so these parameters must be configured. See the example `config.yaml` above for typical values.

Environment variables are mapped to configuration keys (e.g., `SERVER_PORT` maps to `server.port`). To avoid conflicts with other applications, variables can be prefixed with `PRODUCT_SVC_` (e.g., `PRODUCT_SVC_SERVER_PORT`). Both prefixed and non-prefixed variables are supported, but the prefixed version is recommended.

| Parameter                   | Environment Variable                    | Description                                                                           |
|:----------------------------|:----------------------------------------|:--------------------------------------------------------------------------------------|
| `server.port`               | `PRODUCT_SVC_SERVER_PORT`               | The port for the HTTP server to listen on.                                            |
| `server.maxHeaderBytes`     | `PRODUCT_SVC_SERVER_MAXHEADERBYTES`     | The maximum number of bytes the server will read parsing the request headers.         |
| `server.timeout.read`       | `PRODUCT_SVC_SERVER_TIMEOUT_READ`       | The maximum duration for reading the entire request, including the body.              |
| `server.timeout.write`      | `PRODUCT_SVC_SERVER_TIMEOUT_WRITE`      | The maximum duration before timing out writes of the response.                        |
| `server.timeout.idle`       | `PRODUCT_SVC_SERVER_TIMEOUT_IDLE`       | The maximum amount of time to wait for the next request when keep-alives are enabled. |
| `server.timeout.readHeader` | `PRODUCT_SVC_SERVER_TIMEOUT_READHEADER` | The amount of time allowed to read request headers.                                   |
| `database.url`              | `PRODUCT_SVC_DATABASE_URL`              | The connection URL for the PostgreSQL database.                                       |
| `log.level`                 | `PRODUCT_SVC_LOG_LEVEL`                 | The logging level. Can be `debug`, `info`, `warn`, `error`.                           |
| `pprof.enabled`             | `PRODUCT_SVC_PPROF_ENABLED`             | Enables or disables the `pprof` server.                                               |
| `pprof.addr`                | `PRODUCT_SVC_PPROF_ADDR`                | The address for the `pprof` server to listen on (e.g., `localhost:6060`).             |

### Product Service API Endpoints
The API is versioned under /api/v1.

| Method | Endpoint                    | Description                                  |
|:-------|:----------------------------|:---------------------------------------------|
| GET    | /healthz                    | Health check endpoint.                       |
| GET    | /api/v1/products            | Get a paginated list of products.            |
| POST   | /api/v1/products            | Create a new product.                        |
| GET    | /api/v1/products/{id}       | Get a single product by its UUID.            |
| PUT    | /api/v1/products/{id}       | Update a product's details.                  |
| DELETE | /api/v1/products/{id}       | Delete a product by its UUID.                |
| PUT    | /api/v1/products/{id}/stock | Update only the stock quantity of a product. |

### pprof Server

The `pprof` server is a powerful tool for profiling and debugging Go applications. It is disabled by default but can be enabled via configuration.

To enable the `pprof` server, set `pprof.enabled` to `true` in `config.yaml` or use the environment variable `PRODUCT_SVC_PPROF_ENABLED=1`. The server will listen on the address specified by `pprof.addr` (e.g., `localhost:6060`).

Example usage:
```sh
curl http://localhost:6060/debug/pprof/
```

For more information on `pprof`, see the [official documentation](https://pkg.go.dev/net/http/pprof).

## Testing

### Test Types
- **Unit Tests**: Located alongside the code they test. They are self-contained and mock any external dependencies.
- **Integration Tests** (`internal/product/store`): Test the database layer against a real PostgreSQL instance running in a Docker container.
- **E2E Tests** (`internal/tests/e2e`): Test the full application stack by making HTTP requests to the service, which interacts with a test database.

### Skipping Tests
You can skip integration or E2E tests by setting environment variables. This is useful for running only unit tests in environments without Docker.

- Skip integration tests:
  ```sh
  PRODUCT_SVC_SKIP_INTEGRATION_TESTS=1 go test -v ./...
  ```
- Skip E2E tests:
  ```sh
  PRODUCT_SVC_SKIP_E2E_TESTS=1 go test -v ./...
  ```

## Tooling
### Code Generation
This project uses sqlc to generate type-safe Go code for database interactions. If you modify any SQL queries (e.g., in `internal/product/queries`), you must regenerate the Go code. This may need to be done on a per-service basis.

```sh
sqlc generate
```

### Linting
The project is configured with golangci-lint. To run the linter:

```sh
golangci-lint run
```

It is also set up as a pre-commit hook.

### Pre-commit Hooks

This project uses [pre-commit](https://pre-commit.com/) for managing git hooks. The hooks are configured in `.pre-commit-config.yaml` to automatically check and format the code before each commit.

**Local Setup**

1.  Install `pre-commit` (e.g., via pip):
    ```sh
    pip install pre-commit
    ```

2.  Set up the git hook scripts:
    ```sh
    pre-commit install
    ```

Now `pre-commit` will run automatically on `git commit`. You can also run it manually on all files:
```sh
pre-commit run --all-files
```
