.PHONY: dev ci format format-check lint test setup build release clean

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
	# corepack ships with Node 22 and resolves pnpm from the `packageManager`
	# field in package.json, so the version is pinned in exactly one place.
	# The sudo fallback covers images where corepack's shim dir is root-owned.
	corepack enable pnpm 2>/dev/null || sudo corepack enable pnpm
	pnpm install

build: setup
	pnpm build

release: build

clean:
	pnpm clean
