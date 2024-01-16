for i in $(ls -1 docs/images); do
  grep -r --include "*.md" --include "mkdocs*.yml" -m 1 -q $i . || rm docs/images/$i
done