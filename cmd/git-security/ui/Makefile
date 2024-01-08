build:
ifneq ($(BUILD_VERSION),"")
	npm version $(BUILD_VERSION)
endif
	npm install
	npm run generate

clean:
	rm -fr .output node_modules .nuxt
