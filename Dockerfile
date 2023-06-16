###############################################################################
# BUILD STAGE

FROM golang:alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN apk update \
    && apk upgrade \
    && apk add --no-cache git
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' .

###############################################################################
# PACKAGE STAGE

FROM alpine
RUN apk update \
    && apk upgrade --no-cache
EXPOSE 9436
COPY scripts/start.sh /app/
COPY --from=builder /build/mikrotik-exporter /app/
RUN chmod 755 /app/*
WORKDIR /app
ENTRYPOINT ["./start.sh"]