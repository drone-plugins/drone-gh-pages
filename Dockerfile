FROM alpine:3.2

RUN apk update && \
  apk add \
    ca-certificates \
    git \
    openssh \
    curl \
    rsync \
    perl && \
  rm -rf /var/cache/apk/*

ADD drone-gh-pages /bin/
ENTRYPOINT ["/bin/drone-gh-pages"]
