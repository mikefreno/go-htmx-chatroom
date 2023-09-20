# Build the application from source from golang image
FROM golang:latest AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main

# Run the tests in the build stage
RUN go test -v ./...

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /app

# Copy binary file main from build-stage to /app directory
COPY --from=build-stage /app/main /app/main
EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/main"]

