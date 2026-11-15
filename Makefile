GO ?= go
GOFMT ?= gofmt
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# The installed binary runs outside the repository, where .env.local is not
# found, so the Toodledo credentials are embedded like in release builds.
# Environment variables take precedence over .env.local.
dotenv = $(shell sed -n 's/^$(1)=//p' .env.local 2>/dev/null)
TOODLEDO_CLIENT_ID ?= $(call dotenv,TOODLEDO_CLIENT_ID)
TOODLEDO_CLIENT_SECRET ?= $(call dotenv,TOODLEDO_CLIENT_SECRET)

LDFLAGS := -s -w -X main.version=$(VERSION) \
	-X main.clientID=$(TOODLEDO_CLIENT_ID) \
	-X main.clientSecret=$(TOODLEDO_CLIENT_SECRET)

.PHONY: build test check vuln install uninstall clean

# Not echoed, so the client secret stays out of terminal and CI logs.
build:
	@echo 'CGO_ENABLED=0 $(GO) build -trimpath -o bin/tuidledo . (version $(VERSION))'
	@CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/tuidledo .
	@test -n '$(TOODLEDO_CLIENT_ID)' -a -n '$(TOODLEDO_CLIENT_SECRET)' || \
		echo 'Note: no Toodledo credentials embedded; set TOODLEDO_CLIENT_ID and TOODLEDO_CLIENT_SECRET when running.'

test:
	$(GO) test -race ./...

check:
	$(GO) vet ./...
	@test -z "$$($(GOFMT) -l .)" || { echo 'Run gofmt -w .'; exit 1; }
	$(GO) mod tidy -diff
	$(GO) tool staticcheck ./...

# Needs network access to download the Go vulnerability database.
vuln:
	$(GO) tool govulncheck ./...

install: build
	install -Dm755 bin/tuidledo $(DESTDIR)$(BINDIR)/tuidledo

# Keeps config.toml and state.json, including the saved sign-in.
uninstall:
	rm -f $(DESTDIR)$(BINDIR)/tuidledo

clean:
	rm -f bin/tuidledo
