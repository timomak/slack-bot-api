FROM golang:1.22-alpine AS build

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies efficiently
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/slack-bot-api ./cmd/bot

# Use a distroless container for a smaller production image
FROM gcr.io/distroless/static:nonroot

# Copy the binary from the build stage
COPY --from=build /bin/slack-bot-api /app/slack-bot-api

# Set the user to non-root
USER nonroot:nonroot

# Command to run when container starts
ENTRYPOINT ["/app/slack-bot-api"] 