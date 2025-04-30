build:
	@for var in $(shell ls -d */ | grep -v 'util'); do $(MAKE) -C $$var; done
