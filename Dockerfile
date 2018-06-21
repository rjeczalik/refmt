FROM golang:1.10 as builder

WORKDIR /go/src/github.com/rjeczalik/refmt
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build

FROM scratch
COPY --from=builder /go/src/github.com/rjeczalik/refmt/refmt /refmt
ENTRYPOINT ["/refmt"]
