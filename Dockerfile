FROM golang:1.11-alpine3.8 AS build
RUN apk add --no-cache git && \
    rm -rf /var/cache/apk/*
ENV CGO_ENABLED 0
ENV GOOS linux
WORKDIR /build
COPY . .
RUN go build

FROM alpine:3.8
RUN apk add --no-cache ca-certificates && \
    rm -rf /var/cache/apk/*
WORKDIR /
COPY --from=build /build/dscmqtt .
CMD [ "/dscmqtt" ]
