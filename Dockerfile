# step: build go
FROM golang:1.11-alpine3.8 as go-builder

ADD go/src/cmd/go/internal/modload/load.go /usr/local/go/src/cmd/go/internal/modload/

RUN apk add --no-cache --virtual .build-deps \
		bash \
		gcc \
		musl-dev \
		openssl \
		go \
	; \
	export \
# set GOROOT_BOOTSTRAP such that we can actually build Go
		GOROOT_BOOTSTRAP="/usr/lib/go/" \
# ... and set "cross-building" related vars to the installed system's values so that we create a build targeting the proper arch
# (for example, if our build host is GOARCH=amd64, but our build env/image is GOARCH=386, our build needs GOARCH=386)
		GOOS="$(go env GOOS)" \
		GOARCH="$(go env GOARCH)" \
		GOHOSTOS="$(go env GOHOSTOS)" \
		GOHOSTARCH="$(go env GOHOSTARCH)" \
	; \
# also explicitly set GO386 and GOARM if appropriate
# https://github.com/docker-library/golang/issues/184
	apkArch="$(apk --print-arch)"; \
	case "$apkArch" in \
		armhf) export GOARM='6' ;; \
		x86) export GO386='387' ;; \
	esac; \
	\
	cd /usr/local/go/src; \
	./make.bash; \
	\
	rm -rf \
# https://github.com/golang/go/blob/0b30cf534a03618162d3015c8705dd2231e34703/src/cmd/dist/buildtool.go#L121-L125
		/usr/local/go/pkg/bootstrap \
# https://golang.org/cl/82095
# https://github.com/golang/build/blob/e3fe1605c30f6a3fd136b561569933312ede8782/cmd/release/releaselet.go#L56
		/usr/local/go/pkg/obj \
	; \
	apk del .build-deps;


# step: build goproxy
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

COPY --from=go-builder /usr/local/go/bin /usr/local/go/bin
COPY --from=builder /app /app

WORKDIR /app

ENTRYPOINT ["/app/goproxy"]
