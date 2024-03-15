.PHONY: dev
dev:
	yarn dev

.PHONY: ci
ci: setup lint test build

.PHONY: format
format: setup
	yarn format

.PHONY: lints
lint: setup
	yarn lint

.PHONY: test
test: setup
	yarn test

.PHONY: setup
setup:
	yarn install --frozen-lockfile

.PHONY: build
build: setup
	yarn build

.PHONY: release
release: build
