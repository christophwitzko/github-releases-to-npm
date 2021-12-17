FROM golang:1.17

RUN go install github.com/mitchellh/gox@latest

WORKDIR /build
COPY ./build.sh /build

ENTRYPOINT ["/build/build.sh"]
