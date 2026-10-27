# Documentation Makefile.

DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

doc-check-images:   ## Check if all images are used in documentation
	@$(DIR)/resources/bin/check-images.sh

doc-remove-images:  ## Remove unused images from documentation
	@ACTION=remove $(DIR)/resources/bin/check-images.sh

doc-build:          ## Build documentation (used in CI)
	# This command is used to build and deploy a preview to onrender.com
	# Preview URL: https://pmm-doc.onrender.com

	mkdocs build -f $(DIR)/mkdocs.yml

doc-build-pdf: doc-build-image  ## Build documentation in PDF format
	docker run --rm -v $(DIR):/docs -e ENABLED_GIT_REVISION_DATE=false pmm-doc-build \
		bash -c "mkdocs build -f mkdocs-pdf.yml -d /tmp/site && cp /tmp/site/pdf/*.pdf /docs/"

doc-build-image:    ## Build the pmm-doc-build image used by doc-build-pdf and doc-build-preview
	docker build -t pmm-doc-build -f $(DIR)/resources/Dockerfile.build $(DIR)

doc-build-preview: doc-build-image  ## Build documentation and preview at http://localhost:8000
	docker run --rm -p 8000:8000 -v $(DIR):/docs:ro -e ENABLED_GIT_REVISION_DATE=false pmm-doc-build \
		mkdocs serve -f mkdocs.yml --dev-addr=0.0.0.0:8000

doc-search-icons:     ## Search for icons that can be used in documentation
	@open https://squidfunk.github.io/mkdocs-material/reference/icons-emojis/#search
