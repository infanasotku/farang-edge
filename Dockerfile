FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /edge ./cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot AS final

COPY --from=builder --chown=nonroot:nonroot --chmod=755 /edge /bin/edge

USER nonroot

EXPOSE 443

WORKDIR /bin
CMD ["/bin/edge"]
