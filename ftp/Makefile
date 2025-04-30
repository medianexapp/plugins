CHECK_PROGRAM := $(shell which plugin_api 2>/dev/null)
ifeq ($(CHECK_PROGRAM),)
   $(error "Please install go install github.com/medianexapp/plugin_api/cmd/plugin_api@latest")
endif
build:
	plugin_api build
