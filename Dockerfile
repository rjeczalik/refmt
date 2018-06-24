FROM golang:1.10.3 as builder

WORKDIR /go/src/github.com/rjeczalik/refmt
COPY . .
RUN curl -sL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 && \
    chmod +x /usr/local/bin/dep && dep ensure && \
    CGO_ENABLED=0 GOOS=linux go build

FROM scratch
COPY --from=builder /go/src/github.com/rjeczalik/refmt/refmt /refmt
ENTRYPOINT ["/refmt"]
