FROM golang:1.21.9-bullseye AS build-env

ADD . /src
RUN --mount=type=cache,target=/root/.cache/go-build \
      --mount=type=cache,target=/go/pkg/mod \
      cd /src && go build -o /alerts

FROM debian:bullseye

RUN apt-get update \
      && apt-get install -y ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && update-ca-certificates

COPY --from=build-env /alerts /

CMD ["/alerts"]
