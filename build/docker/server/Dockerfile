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

# Disable fastestmirror plugin and update repo URLs
RUN sed -i 's/enabled=1/enabled=0/g' /etc/yum/pluginconf.d/fastestmirror.conf && \
    sed -i -e 's/^\(mirrorlist\)/#\1/g' /etc/yum.repos.d/CentOS-Base.repo && \
    sed -i -e 's/^#baseurl.*/baseurl=http:\/\/vault.centos.org\/centos\/$$releasever\/os\/$$basearch\//g' /etc/yum.repos.d/CentOS-Base.repo

RUN yum -y install epel-release && yum -y install ansible

COPY RPMS /tmp/RPMS
COPY gitCommit /tmp/gitCommit

COPY ansible /opt/ansible
RUN cp -r /opt/ansible/roles /opt/ansible/pmm2-docker/roles
RUN ansible-playbook -vvv -i 'localhost,' -c local /opt/ansible/pmm2-docker/main.yml \
    && ansible-playbook -vvv -i 'localhost,' -c local /usr/share/pmm-update/ansible/playbook/tasks/update.yml \
    && ansible-playbook -vvv -i 'localhost,' -c local /opt/ansible/pmm2/post-build-actions.yml

COPY entrypoint.sh /opt/entrypoint.sh
HEALTHCHECK --interval=3s --timeout=2s --start-period=10s --retries=3 CMD curl -f http://127.0.0.1/v1/readyz || exit 1
CMD ["/opt/entrypoint.sh"]
