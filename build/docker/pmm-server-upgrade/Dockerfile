FROM oraclelinux:9

ARG VERSION
ARG BUILD_DATE

COPY pmm-server-upgrade /bin

LABEL org.label-schema.build-date ${BUILD_DATE}
LABEL org.label-schema.license AGPL-3.0
LABEL org.label-schema.name Percona Monitoring and Management
LABEL org.label-schema.vendor Percona
LABEL org.label-schema.version ${VERSION}
LABEL org.opencontainers.image.created ${BUILD_DATE}
LABEL org.opencontainers.image.licenses AGPL-3.0
LABEL org.opencontainers.image.title Percona Monitoring and Management
LABEL org.opencontainers.image.vendor Percona
LABEL org.opencontainers.image.version ${VERSION}

HEALTHCHECK --interval=3s --timeout=2s --start-period=10s --retries=3 CMD bash -c "[ -S /srv/pmm-server-upgrade.sock ]"
CMD ["pmm-server-upgrade", "run"]
