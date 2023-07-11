FROM --platform=$BUILDPLATFORM golang:1.20-bookworm as build-env

ARG TARGETOS TARGETARCH
ADD . /src
RUN cd /src/auction-bot && GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /main

FROM debian:bookworm

RUN apt-get update \
      && apt-get install -y ca-certificates \
      && rm -rf /var/lib/apt/lists/* \
      && update-ca-certificates

COPY --from=build-env /main /

CMD ["/main"]
