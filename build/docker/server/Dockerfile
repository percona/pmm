FROM centos:7

ARG VERSION
ARG BUILD_DATE

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

# Use COPY as we want to unarchive it with ansible
COPY pmm-client.tar.gz /tmp/

COPY ansible /opt/ansible
RUN cp -r /opt/ansible/roles /opt/ansible/pmm-docker/roles
RUN ansible-playbook -vvv -i 'localhost,' -c local /opt/ansible/pmm-docker/main.yml \
    && ansible-playbook -vvv -i 'localhost,' -c local /usr/share/pmm-update/ansible/playbook/tasks/update.yml \
    && ansible-playbook -vvv -i 'localhost,' -c local /opt/ansible/pmm/post-build-actions.yml

COPY entrypoint.sh /opt/entrypoint.sh
HEALTHCHECK --interval=3s --timeout=2s --start-period=10s --retries=3 CMD curl -f http://127.0.0.1/v1/readyz || exit 1
CMD ["/opt/entrypoint.sh"]
