FROM golang:alpine AS builder

RUN apk add --no-cache git protobuf-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
RUN go install github.com/bufbuild/buf/cmd/buf@latest
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

COPY . .

RUN sqlc generate
RUN buf generate

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/app .

EXPOSE 50051

CMD ["./app"]
