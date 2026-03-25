FROM golang:1.23-alpine

WORKDIR /app

RUN apk add --no-cache bash curl

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN chmod +x /app/entrypoint.sh

EXPOSE 8080

CMD ["sh", "/app/entrypoint.sh"]
