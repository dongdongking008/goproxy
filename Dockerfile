# step: build
FROM golang:1.11-alpine3.8 as builder

COPY . /go/src/goproxy

RUN cd /go/src/goproxy &&\
    CGO_ENABLED=0 GO111MODULE=on go build -o /app/goproxy

COPY go.mod /app

# step: run
FROM golang:1.11-alpine3.8
LABEL maintainer="dongdongking008 <dongdongking008@gmail.com>"

RUN apk --update add openssh-client git && \
    rm -rf /var/cache/apk/*

COPY --from=builder /app /app

WORKDIR /app

ENTRYPOINT ["/app/goproxy"]
