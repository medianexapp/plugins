build:
	@echo ${SERVER_ADDR}
	@echo ${UPLOAD_KEY}
	@echo "{\"server_addr\": \"${SERVER_ADDR}\"}" > util/env.json
	@for var in $(shell ls -d */ | grep -v 'util'); do $(MAKE) -C $$var; done
	@echo "{}" > util/env.json
