
FROM golang:1.15-alpine as BUILDER

WORKDIR /app

COPY . .

RUN go build -o main

# ------------------------------

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main /app

CMD ["./main"]