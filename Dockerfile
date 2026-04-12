FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /xray ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot AS final

COPY --from=builder --chown=nonroot:nonroot --chmod=755 /xray /xray/xray

USER nonroot

EXPOSE 9000

WORKDIR /xray
CMD ["/xray/xray"]
