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
	# Prefer corepack (honors packageManager pin). Fall back to a global npm
	# install so feature builds work against the current rpmbuild image, which
	# ships nodejs+yarn but not pnpm until rpmbuild:3 is republished.
	corepack enable pnpm || sudo corepack enable pnpm || true
	command -v pnpm >/dev/null 2>&1 || sudo npm install -g pnpm@11.1.3
	pnpm install

build: setup
	pnpm build

release: build

clean:
	pnpm clean
