FROM golang:1.20-bookworm as build-env

ADD . /src
RUN cd /src/deputy-claimer && go build -o /main

FROM debian:bookworm

RUN apt-get update \
      && apt-get install -y curl ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && update-ca-certificates

COPY --from=build-env /main /

CMD ["/main"]
