dev:
	yarn dev

ci: setup lint test build

format: setup
	yarn format

lint: setup
	yarn lint

test: setup
	yarn test

setup:
	yarn install --frozen-lockfile

build: setup
	yarn build

release: build

clean:
	rm -r dist
