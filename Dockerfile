FROM alpine:latest
LABEL org.opencontainers.image.source=https://github.com/pteich/crdlens

COPY crdlens /usr/bin/crdlens

ENTRYPOINT ["/usr/bin/crdlens"]