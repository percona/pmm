.PHONY: dev ci format format-check lint check-types test setup build release clean

dev:
	pnpm dev

ci: setup format-check lint check-types test build

format: setup
	pnpm format

format-check: setup
	pnpm format:check

lint: setup
	pnpm lint

# Separate from `lint`, which is oxlint and sees no types, and from `build`, whose
# tsc covers only apps/pmm's own src. Every package already declares `check-types`
# and the turbo task already existed -- nothing ran it, which is how two type errors
# reached a green build (PMM-15664).
check-types: setup
	pnpm check-types

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
