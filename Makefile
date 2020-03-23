REPO_PREFIX := yunion.io/x/yunion-kube
VENDOR_PATH := $(REPO_PREFIX)/vendor
VERSION_PKG_PREFIX := $(VENDOR_PATH)/yunion.io/x/pkg/util/version
ROOT_DIR := $(shell pwd)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin
BUILD_SCRIPT := $(ROOT_DIR)/build/build.sh

GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_VERSION := $(shell git describe --tags --abbrev=14 $(GIT_COMMIT)^{commit})
GIT_TREE_STATE := $(shell s=`git status --porcelain 2>/dev/null`; if [ -z "$$s"  ]; then echo "clean"; else echo "dirty"; fi)
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := "-w \
	-X $(VERSION_PKG_PREFIX).gitBranch=$(GIT_BRANCH) \
	-X $(VERSION_PKG_PREFIX).gitVersion=$(GIT_VERSION) \
	-X $(VERSION_PKG_PREFIX).gitCommit=$(GIT_COMMIT) \
	-X $(VERSION_PKG_PREFIX).gitTreeState=$(GIT_TREE_STATE) \
	-X $(VERSION_PKG_PREFIX).buildDate=$(BUILD_DATE)"

export GO111MODULE:=on
export GOPROXY:=direct
RELEASE_BRANCH:=release/2.12
mod:
	go get yunion.io/x/onecloud@$(RELEASE_BRANCH)
	go get $(patsubst %,%@master,$(shell GO111MODULE=on go mod edit -print | sed -n -e 's|.*\(yunion.io/x/[a-z].*\) v.*|\1|p' | grep -v '/onecloud$$'))
	go mod tidy
	go mod vendor -v

GO_BUILD := go build -mod vendor -ldflags $(LDFLAGS)

CMDS := $(shell find ./cmd -mindepth 1 -maxdepth 1 -type d)

all: build

build: clean fmt
	@for PKG in $(CMDS); do \
		echo build $$PKG; \
		$(GO_BUILD) -o $(BIN_DIR)/`basename $${PKG}` $$PKG; \
	done

bundle:
	./tools/bundle_libraries.sh $(BIN_DIR)/bundles/kube-server $(BIN_DIR)/kube-server

grpc:
	protoc --proto_path=pkg/agent/localvolume --go_out=plugins=grpc:pkg/agent/localvolume \
		localvolume.proto

cmd/%: prepare_dir
	$(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

rpm: build
	$(BUILD_SCRIPT) kube-server
	$(BUILD_SCRIPT) kube-agent

rpmclean:
	rm -rf $(BUILD_DIR)rpms

REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
	   	git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

image: build bundle
	docker build -f Dockerfile -t $(REGISTRY)/kubeserver:$(VERSION) .

image-push: image
	docker push $(REGISTRY)/kubeserver:$(VERSION)

prepare_dir: bin_dir

bin_dir:
	@mkdir -p $(BUILD_DIR)/bin

clean:
	@rm -rf $(BUILD_DIR)

fmt:
	@git ls-files --exclude '*' '*.go' \
		| grep -v '^vendor/' \
		| xargs gofmt -w

swagger-check:
	which swagger || (GO111MODULE=off go get -u github.com/go-swagger/go-swagger/cmd/swagger)

swagger: swagger-check
	GO111MODULE=off swagger generate spec -o ./swagger.yaml --scan-models --work-dir=./pkg/docs

swagger-serve: swagger
	swagger serve -F=swagger swagger.yaml

.PHONY: all build prepare_dir bin_dir clean rpm
