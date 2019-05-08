FROM golang:1-alpine3.9 AS build
RUN apk add --no-cache git && \
    rm -rf /var/cache/apk/*
ENV CGO_ENABLED 0
ENV GOOS linux
WORKDIR /build
COPY . .
RUN go build

FROM alpine:3.9
RUN apk add --no-cache ca-certificates tzdata && \
    rm -rf /var/cache/apk/*
WORKDIR /
COPY --from=build /build/dscmqtt .
CMD [ "/dscmqtt" ]
