
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

ENV PATH_PREFIX=/app
# Copy everything from build-stage to /app directory in our build-release-stage
COPY --from=build-stage /app/main /app/main
COPY --from=build-stage /app/src/scripts /app/src/scripts
COPY --from=build-stage /app/src/styles /app/src/styles
COPY --from=build-stage /app/src/templates /app/src/templates

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/main"]
