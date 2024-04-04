FROM golang:1.20-bullseye as build-env

ADD . /src
RUN cd /src && go build -o /alerts

FROM debian:bullseye

RUN apt-get update \
      && apt-get install -y ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && update-ca-certificates

COPY --from=build-env /alerts /

CMD ["/alerts"]
