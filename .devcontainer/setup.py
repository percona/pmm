#!/usr/bin/python2

# See CONTRIBUTING.md.

from __future__ import print_function, unicode_literals
import os
import subprocess
import time


GO_VERSION = os.getenv("GO_VERSION")
if GO_VERSION is None:
    raise "GO_VERSION is not set"


def run_commands(commands):
    """Runs given shell commands and checks exit codes."""

    for cmd in commands:
        print(">", cmd)
        subprocess.check_call(cmd, shell=True)


def install_packages():
    """Installs required and useful RPM packages."""

    run_commands([
        # to install man pages
        "sed -i '/nodocs/d' /etc/yum.conf",

        # reinstall with man pages
        "yum reinstall -y yum rpm",

        "yum install -y gcc git make pkgconfig glibc-static \
            vim \
            ansible-lint \
            mc tmux psmisc lsof which iproute \
            bash-completion bash-completion-extras \
            man man-pages \
            dh-autoreconf \
            openssl-devel \
            wget"
    ])


def install_go():
    """Installs Go toolchain."""

    run_commands([
        "curl -sS https://raw.githubusercontent.com/travis-ci/gimme/v1.5.4/gimme -o /usr/local/bin/gimme",
        "chmod +x /usr/local/bin/gimme"
    ])

    go_version = str(subprocess.check_output("gimme -r " + GO_VERSION, shell=True).strip())

    if GO_VERSION == "tip":
        run_commands([
            "mkdir $HOME/git_source",
            "wget https://github.com/git/git/archive/refs/tags/v2.34.4.tar.gz -O $HOME/git.tar.gz",
            "tar -xzf $HOME/git.tar.gz -C $HOME/git_source --strip-components 1",
            "cd $HOME/git_source && make configure && ./configure --prefix=/usr && make all && make install",
        ])
        gimme_go_dir = "go"
    else:
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
        "sed -i -e \"s/#listen_addresses = \'localhost\'/listen_addresses = \'*\'/\" /srv/postgres/postgresql.conf",
        "echo 'host    all         all     0.0.0.0/0     trust' >> /srv/postgres/pg_hba.conf",
        "supervisorctl restart postgresql",
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
