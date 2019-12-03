# go run -ldflags "-X mikrotik-exporter/cmd.version=6.6.7-BETA -X mikrotik-exporter/cmd.shortSha=`git rev-parse HEAD`" main.go version

VERSION=`cat VERSION`
SHORTSHA=`git rev-parse --short HEAD`

LDFLAGS=-X main.appVersion=$(VERSION)
LDFLAGS+=-X main.shortSha=$(SHORTSHA)

build:
	go build -ldflags "$(LDFLAGS)" .

utils:
	go get github.com/mitchellh/gox
	go get github.com/tcnksm/ghr

deploy: utils
	CGO_ENABLED=0 gox -os="linux freebsd netbsd" -arch="amd64 arm arm64 386" -parallel=4 -ldflags "$(LDFLAGS)" -output "dist/mikrotik-exporter_{{.OS}}_{{.Arch}}"
	@ghr -t $(GITHUB_TOKEN) -u $(CIRCLE_PROJECT_USERNAME) -r $(CIRCLE_PROJECT_REPONAME) -replace $(VERSION) dist/

dockerhub: deploy
	@docker login -u $(DOCKER_USER) -p $(DOCKER_PASS)
	docker build -t $(CIRCLE_PROJECT_USERNAME)/$(CIRCLE_PROJECT_REPONAME):$(VERSION) .
	docker push $(CIRCLE_PROJECT_USERNAME)/$(CIRCLE_PROJECT_REPONAME):$(VERSION)
	docker build -f Dockerfile.arm64 -t $(CIRCLE_PROJECT_USERNAME)/$(CIRCLE_PROJECT_REPONAME)-linux-arm64:$(VERSION) .
	docker push $(CIRCLE_PROJECT_USERNAME)/$(CIRCLE_PROJECT_REPONAME)-linux-arm64:$(VERSION)
