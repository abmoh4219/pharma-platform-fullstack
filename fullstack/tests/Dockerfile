FROM golang:1.23-alpine

RUN apk add --no-cache bash curl nodejs npm

WORKDIR /workspace

COPY . .

RUN chmod +x /workspace/run_tests.sh /workspace/API_tests/*.sh

CMD ["sh", "-c", "BACKEND_URL=${BACKEND_URL:-http://go-backend:8080} ./run_tests.sh"]
