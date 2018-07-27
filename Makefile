REPO_PREFIX := yunion.io/yunion-kube
VENDOR_PATH := $(REPO_PREFIX)/vendor
VERSION_PKG_PREFIX := $(VENDOR_PATH)/github.com/yunionio/pkg/util/version
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

GO_BUILD := go build -ldflags $(LDFLAGS)

CMDS := $(shell find ./cmd -mindepth 1 -maxdepth 1 -type d)

all: build

build: clean
	@for PKG in $(CMDS); do \
		echo build $$PKG; \
		$(GO_BUILD) -o $(BIN_DIR)/`basename $${PKG}` $$PKG; \
	done

cmd/%: prepare_dir
	$(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

rpm:
	make cmd/$(filter-out $@,$(MAKECMDGOALS))
	$(BUILD_SCRIPT) $(filter-out $@,$(MAKECMDGOALS))

rpmclean:
	rm -rf $(BUILD_DIR)rpms

prepare_dir: bin_dir

bin_dir:
	@mkdir -p $(BUILD_DIR)/bin

clean:
	@rm -rf $(BUILD_DIR)

.PHONY: all build prepare_dir bin_dir clean rpm
