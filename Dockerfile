FROM golang:1.18 as builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o provisioner

FROM alpine:latest
WORKDIR /app
COPY --from=builder /build/provisioner ./
CMD ["./provisioner"]
