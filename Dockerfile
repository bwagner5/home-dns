# Build the manager binary
FROM golang:1.16 as builder

## GOLANG env
ARG GOPROXY="https://proxy.golang.org|direct"
ARG GO111MODULE="on"
ARG CGO_ENABLED=0
ARG GOOS=linux 
ARG GOARCH=amd64 

# Copy go.mod and download dependencies
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build
COPY . .
RUN make build && mv build/home-dns-${GOOS}-${GOARCH} build/home-dns

# Copy the binary into a scratch base image
FROM amazonlinux:2 as amazonlinux
FROM scratch
WORKDIR /
COPY --from=amazonlinux /etc/ssl/certs/ca-bundle.crt /etc/ssl/certs/
COPY --from=builder /app/build/home-dns .
ENTRYPOINT ["/home-dns"]
