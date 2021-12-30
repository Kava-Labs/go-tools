FROM golang:1.17.5-bullseye as build-env

ADD . /src
RUN cd /src/hard-keeper-bot && go build -o /main

FROM debian:bullseye

RUN apt-get update \
      && apt-get install -y ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && update-ca-certificates

COPY --from=build-env /main /

CMD ["/main"]
