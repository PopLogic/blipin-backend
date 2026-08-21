# Build Stage
FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o blipin-backend .


# Run Stage
FROM alpine:3.23.5
WORKDIR /app
COPY --from=builder /app/blipin-backend .
COPY app.env .  
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./db/migration

EXPOSE 8080
CMD ["./blipin-backend"]
ENTRYPOINT ["/app/start.sh"]
