FROM golang:latest as BUILD

WORKDIR /app
COPY . .

RUN go build -o mbserv

FROM debian:stable-slim

RUN apt-get update && apt-get -y install --no-install-recommends ca-certificates curl 

WORKDIR /app

COPY --from=BUILD /app/mbserv .

VOLUME [ "/data" ]

EXPOSE 19132/udp

ENV PATH /app:$PATH

ENTRYPOINT [ "mbserv" ]
