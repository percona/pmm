# This script is used for building a themed site to preview on render.com
# Preview URL: https://pmm-doc.onrender.com

python -m pip install --upgrade pip
pip install wheel

mkdocs build -f ./mkdocs.yml
