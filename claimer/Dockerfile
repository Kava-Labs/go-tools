FROM golang:1.17.5-bullseye as build-env

ADD . /src
# Mount go build and mod caches as container caches, persisted between builder invocations, for faster repeated builds.
RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg/mod cd /src/claimer && go build -o /main

FROM debian:bullseye

RUN apt-get update \
      && apt-get install -y ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && update-ca-certificates

COPY --from=build-env /main /

CMD ["/main"]
