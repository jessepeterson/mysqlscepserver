VERSION = $(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
OSARCH=$(shell go env GOHOSTOS)-$(shell go env GOHOSTARCH)

MYSQLSCEPSERVER=\
	mysqlscepserver-darwin-amd64 \
	mysqlscepserver-darwin-arm64 \
	mysqlscepserver-linux-amd64 \
	mysqlscepserver-linux-arm64 \
	mysqlscepserver-linux-arm

my: mysqlscepserver-$(OSARCH)

docker: mysqlscepserver-linux-amd64

$(MYSQLSCEPSERVER):
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

%-$(VERSION).zip: %.exe
	rm -f $@
	zip $@ $<

%-$(VERSION).zip: %
	rm -f $@
	zip $@ $<

clean:
	rm -f mysqlscepserver-*

release: $(foreach bin,$(MYSQLSCEPSERVER),$(subst .exe,,$(bin))-$(VERSION).zip)

test:
	go test -v -cover -race ./...

.PHONY: my docker $(MYSQLSCEPSERVER) clean release test
