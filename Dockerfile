FROM golang:1.17 AS builder

RUN mkdir /build

COPY . /build

WORKDIR /build

ENV CGO_ENABLED 0

RUN go build -o /lambo ./cmd/lambo

FROM scratch

COPY --from=builder /lambo /lambo

EXPOSE 3000

ENTRYPOINT ["/lambo"]

