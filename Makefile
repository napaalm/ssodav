BINARY := ssodav
VERSION ?= $(shell git describe --always --dirty --tags 2> /dev/null)
URL ?= $(shell git config --get remote.origin.url | sed 's/[a-z0-9]*@//') # remove user from repo's URL

.PHONY: all
all: clean release run

.PHONY: build
build:
	go build -ldflags "-X main.Version=$(VERSION) -X main.SourceURL=$(URL)" -o $(BINARY) cmd/$(BINARY)/main.go

.PHONY: test
test:
	go test -ldflags "-X main.Version=$(VERSION) -X main.SourceURL=$(URL)" ./...

sandbox/config: config | sandbox/
	cp -r $^ $@

.PHONY: sandbox/web
sandbox/web: | sandbox/
	rm -rf $@
	cp -r web $@

sandbox/$(BINARY): build | sandbox/
	cp $(BINARY) $@

.PHONY: sandbox
sandbox: sandbox/web sandbox/config sandbox/$(BINARY)

/tmp/ssodav_ldap.lock:
	echo 'Starting ldap server... (if it does not work add yourself to the group "docker")'
	cd ldap; ./start.sh

.PHONY: ldap
ldap: /tmp/ssodav_ldap.lock

.PHONY: run
run: build sandbox ldap
	cd sandbox; ./$(BINARY)

.PHONY: release
release: linux windows

.PHONY: clean
clean:
	rm -rf sandbox $(BINARY) release

.PHONY: linux
linux:
	mkdir -p release/$(BINARY)-$(VERSION)-$@-amd64
	cp -r config release/$(BINARY)-$(VERSION)-$@-amd64
	cp -r web release/$(BINARY)-$(VERSION)-$@-amd64
	GOOS=$@ GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.SourceURL=$(URL)" -o release/$(BINARY)-$(VERSION)-$@-amd64/ ./...
	cd release; tar -czf $(BINARY)-$(VERSION)-$@-amd64.tar.gz $(BINARY)-$(VERSION)-$@-amd64

.PHONY: windows
windows:
	mkdir -p release/$(BINARY)-$(VERSION)-$@-amd64
	cp -r config release/$(BINARY)-$(VERSION)-$@-amd64
	cp -r web release/$(BINARY)-$(VERSION)-$@-amd64
	GOOS=$@ GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.SourceURL=$(URL)" -o release/$(BINARY)-$(VERSION)-$@-amd64/ ./...
	cd release; zip -qr $(BINARY)-$(VERSION)-$@-amd64.zip $(BINARY)-$(VERSION)-$@-amd64

%/:
	mkdir -p $*
