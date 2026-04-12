FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY api/go.mod api/go.sum* ./
RUN go mod download

COPY api/ .
RUN CGO_ENABLED=1 go build -o server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
ARG GIT_SHA=dev
ENV GIT_SHA=$GIT_SHA

WORKDIR /app
COPY --from=builder /app/server .
COPY static/ ./static/

EXPOSE 8080
ENV PORT=8080
ENV DB_PATH=/data/playground.db
ENV STATIC_DIR=./static

CMD ["./server"]
