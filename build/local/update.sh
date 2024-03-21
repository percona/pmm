#!/bin/bash -e

needs-to-pull() {
  local UPSTREAM=${1:-'@{u}'}
  local LOCAL=$(git rev-parse @)
  local BASE=$(git merge-base @ "$UPSTREAM")
  local REMOTE=$(git rev-parse "$UPSTREAM")

  if [ "$LOCAL" = "$REMOTE" ]; then
    return 1 # false, we are up-to-date
  fi

  if [ "$LOCAL" = "$BASE" ]; then
    return 0 # true, we are behind upstream
  fi
}

rewind() {
  local DIR="$1"
  local BRANCH="$2"

  cd "$DIR"
  CURRENT=$(git branch --show-current)
  git fetch

  if [ "$CURRENT" != "$BRANCH" ]; then
    echo "Currently on $CURRENT, checking out $BRANCH"
    git checkout "$BRANCH"
  fi

  if needs-to-pull; then
    git pull origin
    echo "Submodule has pulled from upstream"
    git logs -n 2
    cd - >/dev/null
    git add "$DIR"
  else
    cd - >/dev/null
    echo "Submodule is up-to-date with upstream"
  fi
}

check-files() {
  local DIR="$1"

  test -z "DIR" && exit 1

  if [ -d "$DIR/sources" ] && [ -f "$DIR/ci-default.yml" ] && [ -f "$DIR/ci.yml" ]; then
    return 0
  fi

  return 1
}

main() {
  local DEPS=
  local CURDIR="$PWD"
  local DIR=pmm-submodules

  # Thouroughly verify the presence of known files, otherwise bail out
  if check-files "."; then # pwd is pmm-submodules
    DIR="."
  elif [ -d "$DIR" ]; then # pwd is outside pmm-submodules
    if ! check-files "$DIR"; then
      echo "FATAL: could not locate known files in ${PWD}/${DIR}"
      exit 1
    fi
  else
    echo "FATAL: could not locate known files in $PWD"
    exit 1
  fi

  cd "$DIR"

  # Join the dependencies from ci-default.yml and ci.yml
  DEPS=$(yq -o=json eval-all '. as $item ireduce ({}; . *d $item )' ci-default.yml ci.yml | jq '.deps')

  echo "This script rewinds submodule branches as per the joint config of 'ci-default.yml' and 'ci.yml'"

  echo "$DEPS" | jq -c '.[]' | while read -r item; do
    branch=$(echo "$item" | jq -r '.branch')
    path=$(echo "$item" | jq -r '.path')
    name=$(echo "$item" | jq -r '.name')
    echo
    echo "Rewinding submodule '$name' ..."
    echo "path: ${path}, branch: ${branch}"

    rewind "$path" "$branch"
  done

  echo
  echo "Printing git status..."
  git status --short
  echo
  echo "Printing git submodule status..."
  git submodule status

  cd "$CURDIR" > /dev/null
}

main
