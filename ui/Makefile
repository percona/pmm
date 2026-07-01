dev:
	pnpm dev

ci: setup format-check lint test build

format: setup
	pnpm format

format-check: setup
	yarn format:check

lint: setup
	pnpm lint

test: setup
	pnpm test

setup:
	pnpm install

build: setup
	pnpm build

release: build

clean:
	pnpm clean
