# go run -ldflags "-X github.com/nshttpd/mikrotik-exporter/cmd.version=6.6.7-BETA -X github.com/nshttpd/mikrotik-exporter/cmd.shortSha=`git rev-parse HEAD`" main.go version

VERSION=`cat VERSION`
SHORTSHA=`git rev-parse --short HEAD`

LDFLAGS=-X github.com/nshttpd/mikrotik-exporter/cmd.version=$(VERSION)
LDFLAGS+=-X github.com/nshttpd/mikrotik-exporter/cmd.shortSha=$(SHORTSHA)

build:
	go build -ldflags "$(LDFLAGS)" .

utils:
	go get github.com/mitchellh/gox
	go get github.com/tcnksm/ghr

deploy: utils
	gox -parallel=4 -ldflags "$(LDFLAGS)" -output "dist/mikrotik-exporter_{{.OS}}_{{.Arch}}"
	ghr -t $(GITHUB_TOKEN) -u $(CIRCLE_PROJECT_USERNAME) -r $(CIRCLE_PROJECT_REPONAME) -replace $(VERSION) dist/