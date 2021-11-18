FROM golang:1.17

RUN go get github.com/mitchellh/gox

WORKDIR /build
COPY ./build.sh /build

ENTRYPOINT ["/build/build.sh"]
