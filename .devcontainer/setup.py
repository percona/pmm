#!/usr/bin/python2

# See CONTRIBUTING.md.

from __future__ import print_function, unicode_literals
import multiprocessing, os, subprocess, time


GO_VERSION = os.getenv("GO_VERSION")
if GO_VERSION is None:
    raise("GO_VERSION is not set")


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
            ansible-lint \
            mc tmux psmisc lsof which iproute \
            bash-completion bash-completion-extras \
            man man-pages",
    ])


def install_go():
    """Installs Go toolchain."""

    run_commands([
        "curl -sS https://raw.githubusercontent.com/travis-ci/gimme/v1.5.4/gimme -o /usr/local/bin/gimme",
        "chmod +x /usr/local/bin/gimme"
    ])

    go_version = subprocess.check_output("gimme -r " + GO_VERSION, shell=True).strip()

    run_commands([
        "gimme " + go_version,
        "rm -fr /usr/local/go",
        "mv -f /root/.gimme/versions/go{go_version}.linux.amd64 /usr/local/go".format(go_version=go_version),
        "update-alternatives --install '/usr/bin/go' 'go' '/usr/local/go/bin/go' 0",
        "update-alternatives --set go /usr/local/go/bin/go",
        "update-alternatives --install '/usr/bin/gofmt' 'gofmt' '/usr/local/go/bin/gofmt' 0",
        "update-alternatives --set gofmt /usr/local/go/bin/gofmt",
        "mkdir -p /root/go/bin",
        "go version",
        "go env"
    ])

def make_install():
    """Runs make install."""

    run_commands([
        "make install",
    ])

def make_init():
    """Runs make init."""

    run_commands([
        "make init",
    ])

def setup():
    """Runs various setup commands."""

    run_commands([
        "supervisorctl stop pmm-managed",
    ])

    # FIXME Remove when https://jira.percona.com/browse/PMM-5197 is done.
    # This is a hack, not a proper solution for this ticket.
    with open("/tmp/setup.sql", "w") as f:
        f.writelines([
            # for run database
            "UPDATE pg_database SET encoding = pg_char_to_encoding('UTF8'), datcollate = 'en_US.utf8', datctype = 'en_US.utf8' WHERE datname = 'pmm-managed';\n",
            # for all future databases, including pmm-managed-dev
            "UPDATE pg_database SET encoding = pg_char_to_encoding('UTF8'), datcollate = 'en_US.utf8', datctype = 'en_US.utf8' WHERE datname = 'template1';\n",
        ])
    run_commands([
        "psql --username=postgres --file=/tmp/setup.sql",
        "psql --username=postgres -l",
        "supervisorctl start pmm-managed",
    ])


def main():
    # install packages early as they will be required below
    install_packages_p = multiprocessing.Process(target=install_packages)
    install_packages_p.start()

    # install Go and wait for it
    install_go()

    # make install (requires make package)
    install_packages_p.join()
    make_init()
    make_install()

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
