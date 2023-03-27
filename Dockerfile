FROM golang:1.19
WORKDIR /usr/src/gophermart
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/ ./...
EXPOSE 8080
CMD ["sh", "-c", "sleep 5 && gophermart"]
ENV RUN_ADDRESS=0.0.0.0:8080 \
    HASH_KEY=abcd \
    ACCRUAL_SYSTEM_ADDRESS=host.docker.internal:8081 \
    DATABASE_URI="postgres://postgres:postgres@host.docker.internal:5432/postgres?sslmode=disable"