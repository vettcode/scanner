# GoReleaser provides the pre-built linux/amd64 binary in the build context.
# debian:12-slim shares glibc with the ubuntu CI runner, ensuring compatibility.
FROM debian:12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    git ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r vettcode \
    && useradd -r -g vettcode -d /home/vettcode -m -s /sbin/nologin vettcode

RUN mkdir -p /home/vettcode/.vettcode/grammars \
    && chown -R vettcode:vettcode /home/vettcode

COPY vettcode /usr/local/bin/vettcode

USER vettcode
WORKDIR /scan

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
LABEL org.opencontainers.image.title="vettcode-scanner" \
      org.opencontainers.image.description="Privacy-first code health scanner for due diligence" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${DATE}" \
      org.opencontainers.image.source="https://github.com/vettcode/scanner" \
      org.opencontainers.image.vendor="VettCode" \
      org.opencontainers.image.licenses="Proprietary"

ENTRYPOINT ["vettcode"]
