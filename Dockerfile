FROM golang:latest as builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o auto-merge-prs main.go

FROM alpine:latest
RUN apk add --no-cache bash curl git jq
RUN curl -fsSL https://github.com/cli/cli/releases/download/v2.0.0/gh_2.0.0_linux_amd64.tar.gz | tar -xz -C /usr/local/bin --strip-components=2 gh_2.0.0_linux_amd64/bin/gh
COPY --from=builder /app/auto-merge /usr/local/bin/auto-merge
LABEL org.opencontainers.image.source="https://github.com/MinhOmega/auto-merge"
ENTRYPOINT ["/usr/local/bin/auto-merge"]
