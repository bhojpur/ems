FROM golang:alpine AS build

RUN apk update && apk add make gcc musl-dev

RUN mkdir -p /go/src/github.com/bhojpur/ems
COPY    .    /go/src/github.com/bhojpur/ems
WORKDIR      /go/src/github.com/bhojpur/ems

RUN ./test.sh
RUN CGO_ENABLED=0 make PREFIX=/opt/ems BLDFLAGS='-ldflags="-s -w"' install


FROM alpine:latest

EXPOSE 4150 4151 4160 4161 4170 4171

RUN mkdir -p /data
WORKDIR      /data

# set up nsswitch.conf for Go's "netgo" implementation
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

# Optional volumes (explicitly configure with "docker run -v ...")
# /data          - used by Bhojpur EMSd for persistent storage across restarts
# /etc/ssl/certs - for SSL Root CA certificates from host

COPY --from=build /opt/ems/bin/ /usr/local/bin/
RUN ln -s /usr/local/bin/*ems* / \
 && ln -s /usr/local/bin/*ems* /bin/