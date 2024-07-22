FROM golang:latest as builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o auto-merge main.go

FROM alpine:latest
COPY --from=builder /app/auto-merge /usr/local/bin/auto-merge
LABEL org.opencontainers.image.source="https://github.com/MinhOmega/auto-merge"
ENTRYPOINT ["/usr/local/bin/auto-merge"]
