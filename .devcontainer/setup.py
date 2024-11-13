#!/usr/bin/python3

# See CONTRIBUTING.md.

import os
import subprocess
import time


GO_VERSION = os.getenv("GO_VERSION")
if GO_VERSION is None:
    raise RuntimeError("GO_VERSION is not set")


def run_commands(commands):
    """Runs given shell commands and checks exit codes."""

    for cmd in commands:
        print(">", cmd)
        subprocess.check_call(cmd, shell=True)


def install_packages():
    """Installs required and useful RPM packages."""

    run_commands([
        "dnf install -y gcc git make pkgconfig \
            vim \
            mc tmux psmisc lsof which iproute diffutils \
            bash-completion \
            man man-pages \
            openssl-devel \
            wget",
        
        "dnf install -y ansible-lint glibc-static --enablerepo=ol9_codeready_builder"

    ])


def install_go():
    """Installs Go toolchain."""

    run_commands([
        "curl -sS https://raw.githubusercontent.com/travis-ci/gimme/v1.5.6/gimme -o /usr/local/bin/gimme",
        "chmod +x /usr/local/bin/gimme"
    ])

    go_version = str(subprocess.check_output("gimme -r " + GO_VERSION, shell=True).strip().decode())

    gimme_go_dir = "go{go_version}.linux.amd64".format(go_version=go_version)

    run_commands([
        "gimme " + go_version,
        "rm -fr /usr/local/go",
        "mv -f /root/.gimme/versions/{gimme_go_dir} /usr/local/go".format(gimme_go_dir=gimme_go_dir),
        "update-alternatives --install '/usr/bin/go' 'go' '/usr/local/go/bin/go' 0",
        "update-alternatives --set go /usr/local/go/bin/go",
        "update-alternatives --install '/usr/bin/gofmt' 'gofmt' '/usr/local/go/bin/gofmt' 0",
        "update-alternatives --set gofmt /usr/local/go/bin/gofmt",
        "mkdir -p /root/go/bin",
        "go version",
        "go env"
    ])


def make_init():
    """Runs make init."""

    run_commands([
        "make init",
    ])


def setup():
    """Runs various setup commands."""
    run_commands([
        # allow connecting from any host, needed to connect from host to PG running in docker
        "sed -i -e \"s/#listen_addresses = \'localhost\'/listen_addresses = \'*\'/\" /srv/postgres14/postgresql.conf",
        # Turns fsync off. Create database operations with fsync on are very slow on Ubuntu.
        # Having fsync off in dev environment is fine.
        "sed -i -e \"s/#fsync = on/fsync = off/\" /srv/postgres14/postgresql.conf",
        "echo 'host    all         all     0.0.0.0/0     trust' >> /srv/postgres14/pg_hba.conf",
        # "supervisorctl restart postgresql",
    ])


def main():
    install_packages()
    install_go()
    make_init()

    # do basic setup
    setup()


MARKER = "/tmp/devcontainer-setup-done"
if os.path.exists(MARKER):
    print(MARKER, "exists, exiting.")
    exit(0)

start = time.time()
main()
print("Done in", time.time() - start)

open(MARKER, 'w').close()
