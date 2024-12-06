#!/bin/bash -e

git config --global --add safe.directory /app

# Join the dependencies from gitmodules.yml and ci.yml
rm -f gitmodules.yml
python ci.py --convert

DEPS=$(yq -o=json eval-all '. as $item ireduce ({}; . *d $item )' gitmodules.yml ci.yml | jq '.deps')
DEPS=$(echo "$DEPS" | jq -r '[.[] | {name: .name, branch: .branch, path: .path, url: .url}]')
rm -f gitmodules.yml
echo "$DEPS" > /app/build/build.json
echo -n > /tmp/deps.txt

echo "$DEPS" | jq -c '.[]' | while read -r item; do
  branch=$(echo "$item" | jq -r '.branch')
  path=$(echo "$item" | jq -r '.path')
  name=$(echo "$item" | jq -r '.name')
  echo "name=${name}:path=${path}:branch=${branch}" >> /tmp/deps.txt
done

mv -f /tmp/deps.txt /app/build/deps.txt
