FROM golang:1.23-alpine

RUN apk add --no-cache bash curl nodejs npm
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /workspace

COPY . .

RUN chmod +x /workspace/run_tests.sh /workspace/API_tests/*.sh
RUN cd /workspace/backend && go mod download
RUN cd /workspace/frontend && npm ci --no-audit --no-fund

CMD ["sh", "-c", "BACKEND_URL=${BACKEND_URL:-http://go-backend:8080} ./run_tests.sh"]
