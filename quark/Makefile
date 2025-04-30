CHECK_PROGRAM := $(shell which plugin_api 2>/dev/null)
ifeq ($(CHECK_PROGRAM),)
   $(error "plugin_api not found,install cmd: go install github.com/medianexapp/plugin_api/cmd/plugin_api@latest")
endif
build:
			plugin_api build
