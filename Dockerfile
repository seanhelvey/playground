FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY api/go.mod api/go.sum* ./
RUN go mod download

COPY api/ .
RUN CGO_ENABLED=1 go build -o server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
COPY data.json ./data.json
COPY tasks.json ./tasks.json
COPY static/ ./static/

EXPOSE 8080
ENV PORT=8080
ENV DB_PATH=/data/playground.db
ENV SEED_PATH=./data.json
ENV STATIC_DIR=./static

CMD ["./server"]
