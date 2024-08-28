FROM golang:1.21.9-bookworm as build-env

ADD . /src
RUN --mount=type=cache,target=/root/.cache/go-build \
      --mount=type=cache,target=/go/pkg/mod \
      cd /src/auction-bot && go build -o /main

FROM debian:bookworm

RUN apt-get update \
      && apt-get install -y ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && update-ca-certificates

COPY --from=build-env /main /

CMD ["/main"]
