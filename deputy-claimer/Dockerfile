FROM golang:1.15.6-buster as build-env

ADD . /src
RUN cd /src/deputy-claimer && go build -o /main

FROM debian:buster

RUN apt-get update \
      && apt-get install -y ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && update-ca-certificates

COPY --from=build-env /main /

CMD ["/main"]
