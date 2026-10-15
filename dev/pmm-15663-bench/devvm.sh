#!/bin/bash
# Runs PMM's dev container on CHAOS VM PMM-15663-dev for a worktree on the runner, which has no Go
# toolchain and must not run PostgreSQL. Run it from inside the worktree.
#
#   devvm.sh setup          clone the branch on the VM and start the dev container (once)
#   devvm.sh sync           make the VM's tree equal to the local working tree (new files included)
#   devvm.sh run '<cmd>'    run a shell command as root in the container, at the repo root
#   devvm.sh make <target>  run a make target as root in the container (the .devcontainer Makefile)
#   devvm.sh pull           bring files changed on the VM since the last sync (e.g. by make gen) back
#
# sync marks untracked files intent-to-add (git add -N) so they travel; .gitignore'd files never do.
set -euo pipefail

RUN=${DEVVM_RUN:-PMM-15663-dev}
BRANCH=${DEVVM_BRANCH:-PMM-15663-scoped-annotations}
VMSH=/opt/pmm-ai/plugins/pmm-qa/skills/chaos-docker-provisioning/scripts/chaos-vm.sh
REPO=/root/pmm
CDIR=/root/go/src/github.com/percona/pmm

in_container() {
	"$VMSH" ssh "$RUN" "docker exec --workdir=$CDIR --user root pmm-server bash -lc $(printf '%q' "$1")"
}

case "${1:-}" in
setup)
	"$VMSH" ssh "$RUN" 'bash -s' <<EOF
set -e
command -v make >/dev/null || dnf -y -q install make git 2>/dev/null || (apt-get -qq update && apt-get -qq install -y make git)
if [ ! -d $REPO ]; then
	git clone -q --branch $BRANCH https://github.com/percona/pmm.git $REPO
	git -C $REPO rev-parse HEAD > /root/.devvm-base
fi
cd $REPO
git config user.email devvm@localhost
git config user.name devvm
[ -f .env ] || cp .env.dev.example .env
make env-up
docker exec -u root pmm-server bash -lc "git config --global --add safe.directory $CDIR"
docker exec --workdir=$CDIR --user root pmm-server make init
echo DEVVM_READY base=\$(cat /root/.devvm-base)
EOF
	;;
sync)
	base=$("$VMSH" ssh "$RUN" cat /root/.devvm-base)
	git add -A -N
	git diff --binary "$base" | "$VMSH" ssh "$RUN" "cd $REPO && git reset -q --hard $base && git clean -qfd && git apply --allow-empty --whitespace=nowarn && git add -A && git commit -qm devvm-sync --allow-empty && echo synced to \$(git rev-parse --short HEAD) on base ${base:0:9}"
	;;
run)
	in_container "$2"
	;;
make)
	shift
	in_container "make $(printf '%q ' "$@")"
	;;
pull)
	patch=$(mktemp)
	"$VMSH" ssh "$RUN" "cd $REPO && git add -A && git diff --cached --binary HEAD" > "$patch"
	if [ -s "$patch" ]; then
		git apply --whitespace=nowarn "$patch"
		git apply --numstat "$patch" | awk '{print "pulled " $3}'
	else
		echo "nothing to pull"
	fi
	rm -f "$patch"
	;;
*)
	sed -n '2,12p' "$0"
	exit 2
	;;
esac
