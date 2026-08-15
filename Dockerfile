FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/vexbridge ./cmd/vexbridge

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/vexbridge /vexbridge
ENTRYPOINT ["/vexbridge"]
