# GoCommerce
[![Go Report Card](https://goreportcard.com/badge/github.com/abgdnv/gocommerce)](https://goreportcard.com/report/github.com/abgdnv/gocommerce)
![CI](https://github.com/abgdnv/gocommerce/actions/workflows/go.yml/badge.svg)
[![License](https://img.shields.io/badge/license-MIT-blue)](https://github.com/abgdnv/gocommerce/blob/main/LICENSE)

A foundational e-commerce demo project built on a Go-based microservices architecture. This repository serves as a template for building production-ready services, demonstrating best practices in project structure, configuration, database interaction, and testing.

This repository will eventually host multiple services (products, orders, etc.), starting with the `product_service`.

## Features (Product Service)

-   **RESTful & gRPC APIs**: Provides both a clean RESTful API for external clients and a high-performance gRPC API for internal service-to-service communication.
-   **Logging**: Uses `slog` for structured, context-aware logging.
-   **Configuration**: Employs a shared config loader (`koanf`) to read settings from YAML files, `.env`, and environment variables.
-   **Database Layer**: Integrates with PostgreSQL using `pgx` (driver), `sqlc` (type-safe code generation), and `golang-migrate` (schema migrations).
-   **Project Structure**: Adheres to Go's standard project layout, with a clean separation of concerns.
-   **Dockerized Environment**: Includes a `docker-compose.yml` for easy local setup of the service and its PostgreSQL database.
-   **Graceful Shutdown**: Implements graceful server shutdown for both HTTP and gRPC servers to handle in-flight requests.
-   **Testing**:
    -   Unit tests with mocks (`testify/mock`).
    -   Integration tests against a real database using `testcontainers-go`.
    -   End-to-end (E2E) tests for the public API.
-   **Profiling Ready**: Optional `pprof` server for performance analysis, configurable and disabled by default.
-   **Code Quality**: Enforced via `golangci-lint` and `pre-commit` hooks.

## Prerequisites

-   [Go](https://golang.org/dl/) (version 1.24 or newer)
-   [Docker](https://www.docker.com/get-started) and [Docker Compose](https://docs.docker.com/compose/install/)
-   [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (for regenerating database code)
-   [golang-migrate](https://github.com/golang-migrate/migrate) (for running database migrations)

## Getting Started

The following instructions are for running the `product_service`.

### 1. Clone the Repository

```sh
git clone https://github.com/abgdnv/gocommerce.git
cd gocommerce
```

### 2. Configure the Environment

Create a `.env` file from the example. This file holds the configuration for Docker Compose and the services.

```sh
cp example.env .env
```
*You can modify `.env` if you need to change ports or credentials.*

### 3. Run with Docker Compose

Build and start the services in detached mode:

```sh
docker-compose up --build -d
```
*This command starts both the `product_service` and its PostgreSQL database. All databases are created on the first start of the database container, and migrations are applied automatically every time the stack is started.*

---

### Development

For local development, you might want to run the Go application directly on your host.

1.  **Start only the database:**
    ```sh
    docker-compose up -d db
    ```
2.  **Run migrations:**

    You'll need `golang-migrate` installed locally. Don't forget to create database if it doesn't exist.
    ```sh
    migrate -path internal/product/store/migrations -database "postgres://<user>:<password>@<databaseb host>:5432/<database>?sslmode=disable" up
    ```
3.  **Run the application:**
    ```sh
    go run ./cmd/product_service/
    ```

### Configuration

The application is configured with the following precedence:
1.  **Environment variables** (prefixed with `PRODUCT_SVC_`, e.g., `PRODUCT_SVC_SERVER_PORT=8081`).
2.  **`.env` file**.
3.  **`configs/product_service.yaml`** file.

All configuration options are defined in the `internal/product/config/config.go` struct and validated on startup.

#### Configuration Parameters

The following parameters can be set in `configs/product_service.yaml` or via environment variables (which take precedence). The application does not have built-in default values, so these parameters must be configured.


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
| `grpc.port`                 | `PRODUCT_SVC_GRPC_PORT`                 | The port for the gRPC server to listen on.                                            |
| `grpc.reflection`           | `PRODUCT_SVC_GRPC_REFLECTION`           | Enables gRPC reflection.                                                              |

### API Endpoints (Product Service)

#### REST API

| Method | Endpoint                    | Description                                  |
|:-------|:----------------------------|:---------------------------------------------|
| GET    | /healthz                    | Health check endpoint.                       |
| GET    | /api/v1/products            | Get a paginated list of products.            |
| POST   | /api/v1/products            | Create a new product.                        |
| GET    | /api/v1/products/{id}       | Get a single product by its UUID.            |
| PUT    | /api/v1/products/{id}       | Update a product's details.                  |
| DELETE | /api/v1/products/{id}       | Delete a product by its UUID.                |
| PUT    | /api/v1/products/{id}/stock | Update only the stock quantity of a product. |

#### gRPC API

The service exposes a gRPC API for internal communication. You can interact with it using `grpcurl`.

```sh
# List available gRPC services
grpcurl -plaintext localhost:50051 list

# Call the GetProduct method
grpcurl -plaintext -d '{"id": "<id>"}' localhost:50051 product.v1.ProductService/GetProduct
```

### pprof Server

The `pprof` server is a powerful tool for profiling and debugging Go applications. It is disabled by default but can be enabled via configuration.

To enable the `pprof` server, set `pprof.enabled` to `true` in `config.yaml` or use the environment variable `PRODUCT_SVC_PPROF_ENABLED=1`. The server will listen on the address specified by `pprof.addr` (e.g., `localhost:6060`).

Example usage:
```sh
curl http://localhost:6060/debug/pprof/
```

For more information on `pprof`, see the [official documentation](https://pkg.go.dev/net/http/pprof).

## Testing

The project includes a multi-layered testing strategy. You can run all tests with:
```sh
go test -v ./...
```
-   **Unit Tests**: Located alongside the code (`*_test.go`).
-   **Integration Tests**: Found in `internal/product/store/store_integration_test.go`, use `testcontainers-go` to spin up a real PostgreSQL container for testing the database layer.
-   **E2E Tests**: Found in `internal/product/tests/e2e/`, test the full application by making HTTP requests.

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

### Code Generation (`sqlc`)

If you modify any SQL queries in `internal/product/store/queries/`, you must regenerate the Go code.

```sh
 make sqlc.gen
 ```

### Code Generation (gRPC - Protocol Buffers)

If you change anything in `proto/product/v1/product.proto`, run the following command from the project root:

```sh
make proto.gen
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

Now, hooks will run automatically on `git commit`. To run them manually:
```sh
pre-commit run --all-files
```
