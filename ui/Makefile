dev:
	pnpm dev

ci: setup format-check lint test build

format: setup
	pnpm format

format-check: setup
	pnpm format:check

lint: setup
	pnpm lint

test: setup
	pnpm test

setup:
	corepack enable pnpm || sudo corepack enable pnpm
	pnpm install

build: setup
	pnpm build

release: build

clean:
	pnpm clean
