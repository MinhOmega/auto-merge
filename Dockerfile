# Start from the latest Go base image
FROM golang:latest as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o auto-merge main.go

# Start a new stage from scratch
FROM alpine:latest

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/auto-merge /usr/local/bin/auto-merge

# Add label with source information
LABEL org.opencontainers.image.source="https://github.com/MinhOmega/auto-merge"

# Command to run the executable
ENTRYPOINT ["/usr/local/bin/auto-merge"]
