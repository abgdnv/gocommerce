# --- Builder Stage ---
# Use a specific version of the Go image for reproducibility.
# Alpine version is used to keep the builder image size small.
# 'AS builder' names this stage, so we can refer to it in the final stage.
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container.
WORKDIR /src

# Copy go.mod and go.sum files to download dependencies.
# This is done separately to leverage Docker's layer caching.
# If these files don't change, the dependencies layer won't be rebuilt on subsequent builds.
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY product_service/go.mod product_service/go.sum ./product_service/

# Download Go modules.
WORKDIR /src/product_service
RUN go mod download
WORKDIR /src

# Copy the rest of the source code into the container.
COPY pkg/ ./pkg/
COPY product_service/ ./product_service/

# Build the Go application.
# CGO_ENABLED=0 creates a static binary without C dependencies, which is ideal for containers.
# GOOS=linux and GOARCH=amd64 specify the target OS and architecture for the final image.
# -ldflags="-w -s" strips debug information, reducing the binary size.
WORKDIR /src/product_service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o app ./cmd/

# --- Final Stage ---
# Use a minimal base image for the final container to reduce its size and attack surface.
FROM alpine:3.22

# Set the working directory.
WORKDIR /app

# Copy only the compiled application binary from the builder stage.
COPY --from=builder /src/product_service/app .

# Create a non-root user for security purposes.
# -D: create a system user (no password, no home dir).
# Then, change the ownership of the application to this new user.
RUN adduser -D -g '' appuser && chown appuser:appuser /app/app
# Switch to the non-root user. Running as a non-root user is a security best practice.
USER appuser

# Expose the port the application will run on.
# This is for documentation purposes; it doesn't publish the port to the host.
EXPOSE 8080

# The command to run when the container starts.
CMD ["./app"]
