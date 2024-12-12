doc-check-images:   ## Check if all images are used in documentation
	@documentation/resources/bin/check-images.sh

doc-remove-images:  ## Remove unused images from documentation
	@ACTION=remove documentation/resources/bin/check-images.sh

doc-build:          ## Build documentation
	# This command is used to build and deploy a preview to onrender.com
	# Preview URL: https://pmm-doc.onrender.com

	# python -m pip install --upgrade pip
	# pip install wheel

	mkdocs build -f ./mkdocs.yml
