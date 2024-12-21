# Documentation Makefile.

DOCDIR := documentation

ifeq ($(shell basename $(CURDIR)),$(DOCDIR))
	DIR := $(CURDIR)
else
	DIR := $(CURDIR)/$(DOCDIR)
endif

doc-check-images:   ## Check if all images are used in documentation
	@$(DIR)/resources/bin/check-images.sh

doc-remove-images:  ## Remove unused images from documentation
	@ACTION=remove $(DIR)/resources/bin/check-images.sh

doc-build:          ## Build documentation (used in CI)
	# This command is used to build and deploy a preview to onrender.com
	# Preview URL: https://pmm-doc.onrender.com

	mkdocs build -f $(DIR)/mkdocs.yml

doc-build-pdf:      ## Build documentation in PDF format
	docker run --rm --platform=linux/amd64 -v $(DIR):/docs -w /build perconalab/pmm-doc-md:latest \
		bash -c " \
			cp -r /docs/* /build/ && \
			git init && \
			git config user.email 'doc-team@percona.com' && \
			git add --all && \
			git commit -am 'Initial commit' > /dev/null && \
			mkdocs build -f mkdocs-pdf.yml && \
			cp /build/site/pdf/*.pdf /docs/ \
		"

doc-build-image:    ## Build perconalab/pmm-doc-md:latest image
	# @docker buildx build --platform=linux/amd64 --progress=plain -t perconalab/pmm-doc-md:latest -f documentation/resources/Dockerfile.build .

doc-view-icons:     ## View icons that can be used in documentation
	@open https://squidfunk.github.io/mkdocs-material/reference/icons-emojis/#search
