FROM centos:7

ARG VERSION
ARG BUILD_DATE

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

EXPOSE 80 443

WORKDIR /opt

RUN yum -y install epel-release && yum -y install ansible

COPY RPMS /tmp/RPMS
COPY gitCommit /tmp/gitCommit

COPY ansible /opt/ansible
RUN cp -r /opt/ansible/roles /opt/ansible/pmm2-docker/roles
RUN ansible-playbook -vvv -i 'localhost,' -c local /opt/ansible/pmm2-docker/main.yml \
    && ansible-playbook -vvv -i 'localhost,' -c local /usr/share/pmm-update/ansible/playbook/tasks/update.yml \
    && ansible-playbook -vvv -i 'localhost,' -c local /opt/ansible/pmm2/post-build-actions.yml

RUN cp /usr/share/pmm-server/entrypoint.sh /opt/entrypoint.sh
HEALTHCHECK --interval=3s --timeout=2s --start-period=10s --retries=3 CMD curl -f http://127.0.0.1/v1/readyz || exit 1
CMD ["/opt/entrypoint.sh"]
