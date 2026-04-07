FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /xray

FROM gcr.io/distroless/static-debian12:nonroot AS final

COPY --from=builder --chown=nonroot:nonroot --chmod=755 /xray /xray/xray
COPY --from=builder --chown=nonroot:nonroot --chmod=755 /app/config.json /xray/config.json

USER nonroot

EXPOSE 9000

WORKDIR /xray
CMD ["/xray/xray"]
