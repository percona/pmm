#!/bin/sh

set -o errexit
set -o xtrace

root_is_needed='no'

check_command() {
    command -v "$@" > /dev/null 2>&1
}

run_root() {
    sh='sh -c'
    if [ "$(id -un)" != 'root' ]; then
        if check_command sudo; then
            sh='sudo -E sh -c'
        elif check_command su; then
            sh='su -c'
        else
            echo ERROR: root rights needed to run "$*" command
            exit 1
        fi
    fi
    ${sh} "$@"
}

install_docker() {
    if ! check_command docker; then
        echo Installing docker
        curl -fsSL get.docker.com -o /tmp/get-docker.sh \
            || wget -qO /tmp/get-docker.sh get.docker.com
        sh /tmp/get-docker.sh
        run_root 'service docker start' || :
    fi
    if ! docker ps; then
        root_is_needed='yes'
        if ! run_root 'docker ps'; then
            echo ERROR: cannot run "docker ps" command
            exit 1
        fi
    fi
}

run_docker() {
    if [ "${root_is_needed}" = 'yes' ]; then
        run_root "docker $*"
    else
        sh -c "docker $*"
    fi
}


start_pmm() {
    run_docker pull percona/pmm-server:latest

    if ! run_docker inspect pmm-data >/dev/null; then
        run_docker create \
            -v /opt/prometheus/data \
            -v /opt/consul-data \
            -v /var/lib/mysql \
            -v /var/lib/grafana \
            --name pmm-data \
            percona/pmm-server:latest /bin/true
    fi

    if run_docker inspect pmm-server >/dev/null; then
        run_docker stop pmm-server || :
        run_docker rename pmm-server "pmm-server-$(date "+%F-%H%M%S")"
    fi

    run_docker run -d \
        -p 80:80 \
        --volumes-from pmm-data \
        --name pmm-server \
        --restart always \
        percona/pmm-server:latest
}

main() {
    install_docker
    start_pmm
}

main
exit 0
