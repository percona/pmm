#! env bash

CMD="bin/command-file.txt"
cat source/.res/replace.txt| perl bin/remove-replacements.pl > $CMD
# Run the sed on each .rst file
find . -name "*.rst" -exec sed -i '' -f $CMD {} \;
# Replacements also in text includes
find source/.res/contents -name "*.txt" -exec sed -i '' -f $CMD {} \;
# Remove include for replace.txt
find . -name "*.rst" -exec sed -i '' 's/^.. include::.*replace.txt$//g' {} \;