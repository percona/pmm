ARG POSTGRES_IMAGE
FROM $POSTGRES_IMAGE

COPY . .
COPY queries/ /docker-entrypoint-initdb.d/
RUN chown postgres:postgres /certs/* /conf
