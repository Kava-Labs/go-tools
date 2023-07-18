FROM debian:bookworm

RUN apt-get update && apt-get install -y ca-certificates wget moreutils jq curl unzip

ARG BVER=v0.10.15

# Contains bnbchaind and bnbcli
RUN wget -O /tmp/linux_binary.zip https://github.com/binance-chain/node/releases/download/${BVER}/linux_binary.zip && \
    unzip -j /tmp/linux_binary.zip linux/* -d /usr/local/bin && \
    rm /tmp/linux_binary.zip

RUN chmod u+x /usr/local/bin/bnbchaind /usr/local/bin/bnbcli
