FROM golang:1.23-alpine
LABEL maintainer="Thornton Phillis (dev@th0rn0.co.uk)"

WORKDIR /app

ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
RUN apk add --no-cache gcc musl-dev

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ .

RUN go build -ldflags='-s -w -extldflags "-static"' -o bin/stream-proxy ./cmd/stream-proxy

EXPOSE 8080

CMD [ "bin/stream-proxy" ]