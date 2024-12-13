doc-check-images:   ## Check if all images are used in documentation
	@documentation/resources/bin/check-images.sh

doc-remove-images:  ## Remove unused images from documentation
	@ACTION=remove documentation/resources/bin/check-images.sh

DOCDIR := documentation

ifeq ($(shell basename $(CURDIR)),$(DOCDIR))
	CONFIG_FILE := $(CURDIR)/mkdocs.yml
else
	CONFIG_FILE := $(CURDIR)/$(DOCDIR)/mkdocs.yml
endif

doc-build:          ## Build documentation
	# This command is used to build and deploy a preview to onrender.com
	# Preview URL: https://pmm-doc.onrender.com

	mkdocs build -f $(CONFIG_FILE)
