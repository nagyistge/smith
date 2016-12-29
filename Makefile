VERSION_VAR := main.Version
GIT_VAR := main.GitCommit
BUILD_DATE_VAR := main.BuildDate
REPO_VERSION := "0.0"
#REPO_VERSION = $$(git describe --abbrev=0 --tags)
BUILD_DATE := $$(date +%Y-%m-%d-%H:%M)
GIT_HASH := $$(git rev-parse --short HEAD)
GOBUILD_VERSION_ARGS := -ldflags "-s -X $(VERSION_VAR)=$(REPO_VERSION) -X $(GIT_VAR)=$(GIT_HASH) -X $(BUILD_DATE_VAR)=$(BUILD_DATE)"
BINARY_NAME := smith
IMAGE_NAME := atlassian/smith
ARCH ?= darwin
GOVERSION := 1.7
GP := /gopath
MAIN_PKG := github.com/atlassian/smith/cmd/smith

setup-minikube-ci:
# Install libvirt and qemu-kvm on your system, e.g.
	sudo apt-get update -qq
	sudo apt-get install -qq libvirt-bin qemu-kvm
# Add yourself to the libvirtd group (may vary by linux distro) so you don't need to sudo
	sudo usermod -a -G libvirtd $$USER

	curl -L https://github.com/docker/machine/releases/download/v0.8.2/docker-machine-$$(uname -s)-$$(uname -m) -o docker-machine
	chmod +x docker-machine
	sudo mv docker-machine /usr/local/bin/docker-machine

	curl -L https://github.com/dhiltgen/docker-machine-kvm/releases/download/v0.7.0/docker-machine-driver-kvm -o docker-machine-driver-kvm
	chmod +x docker-machine-driver-kvm
	sudo mv docker-machine-driver-kvm /usr/local/bin/docker-machine-driver-kvm

	docker-machine version

	curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
	chmod +x minikube
	./minikube version
	./minikube config set WantReportErrorPrompt false

setup-ci:
	go get -u github.com/Masterminds/glide
	go get -u golang.org/x/tools/cmd/goimports
	glide install --strip-vendor

build: *.go fmt
	go build -o build/bin/$(ARCH)/$(BINARY_NAME) $(GOBUILD_VERSION_ARGS) $(MAIN_PKG)

build-race: *.go fmt
	go build -race -o build/bin/$(ARCH)/$(BINARY_NAME) $(GOBUILD_VERSION_ARGS) $(MAIN_PKG)

build-all:
	go build $$(glide nv)

fmt:
	gofmt -w=true -s $$(find . -type f -name '*.go' -not -path "./vendor/*")
	goimports -w=true -d $$(find . -type f -name '*.go' -not -path "./vendor/*")

minikube-test:
	KUBERNETES_SERVICE_HOST="$$(minikube ip)" \
	KUBERNETES_SERVICE_PORT=8443 \
	KUBERNETES_CA_PATH="$$HOME/.minikube/ca.crt" \
	KUBERNETES_CLIENT_CERT="$$HOME/.minikube/apiserver.crt" \
	KUBERNETES_CLIENT_KEY="$$HOME/.minikube/apiserver.key" \
	go test -race -v $$(glide nv)

test-race:
	go test -race $$(glide nv)

# Compile a static binary. Cannot be used with -race
docker:
	docker run \
		--rm \
		-v "$(GOPATH)":"$(GP)" \
		-w "$(GP)/src/github.com/atlassian/smith" \
		-e GOPATH="$(GP)" \
		-e CGO_ENABLED=0 \
		golang:$(GOVERSION) \
		go build -o build/bin/linux/$(BINARY_NAME) $(GOBUILD_VERSION_ARGS) -a -installsuffix cgo $(MAIN_PKG)
	docker build --pull -t $(IMAGE_NAME):$(GIT_HASH) build

# Compile a binary with -race. Needs to be run on a glibc-based system.
docker-race:
	docker run \
		--rm \
		-v "$(GOPATH)":"$(GP)" \
		-w "$(GP)/src/github.com/atlassian/smith" \
		-e GOPATH="$(GP)" \
		golang:$(GOVERSION) \
		go build -race -o build/bin/linux/$(BINARY_NAME) $(GOBUILD_VERSION_ARGS) -a -installsuffix cgo $(MAIN_PKG)
	docker build --pull -t $(IMAGE_NAME):$(GIT_HASH)-race -f build/Dockerfile-glibc build

release-hash: docker
	docker push $(IMAGE_NAME):$(GIT_HASH)

release-normal: release-hash
#	docker tag $(IMAGE_NAME):$(GIT_HASH) $(IMAGE_NAME):latest
#	docker push $(IMAGE_NAME):latest
	docker tag $(IMAGE_NAME):$(GIT_HASH) $(IMAGE_NAME):$(REPO_VERSION)
	docker push $(IMAGE_NAME):$(REPO_VERSION)

release-hash-race: docker-race
	docker push $(IMAGE_NAME):$(GIT_HASH)-race

release-race: docker-race
	docker tag $(IMAGE_NAME):$(GIT_HASH)-race $(IMAGE_NAME):$(REPO_VERSION)-race
	docker push $(IMAGE_NAME):$(REPO_VERSION)-race

release: release-normal release-race

.PHONY: build
