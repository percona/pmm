BINARY := webhook
GO     ?= go

.PHONY: $(MAKECMDGOALS)

build:
	$(GO) build -trimpath -ldflags='-s -w' -o $(BINARY) .

test:
	$(GO) test ./... -count=1

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

run:
	$(GO) run .

clean:
	rm -f $(BINARY)
