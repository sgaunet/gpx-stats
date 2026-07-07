# Minimal, secure image. The CA-cert stage doubles as the documented
# minimal-base fallback when FROM scratch is not feasible (US3 AS3).
FROM --platform=$BUILDPLATFORM alpine:3.24 AS certs
RUN apk add --no-cache ca-certificates

FROM scratch
# GoReleaser's dockers_v2 stages the pre-built binary under $TARGETPLATFORM/.
ARG TARGETPLATFORM
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ${TARGETPLATFORM}/gpx-stats /usr/local/bin/gpx-stats
USER 65534:65534
ENTRYPOINT ["/usr/local/bin/gpx-stats"]
