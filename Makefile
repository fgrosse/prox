VERSION=$(shell git describe --tags)

.PHONY: release
release:
	mkdir -p releases
	mkdir -p release-$(VERSION)
	cp LICENSE release-$(VERSION)
	cp README.md release-$(VERSION)

	# Linux 64
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o release-$(VERSION)/prox ./cmd/prox
	tar -czf prox-$(VERSION)-linux64.tar.gz -C release-$(VERSION) .
	mv prox-$(VERSION)-*.tar.gz releases

	# Linux 32
	GOOS=linux GOARCH=386 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o release-$(VERSION)/prox ./cmd/prox
	tar -czf prox-$(VERSION)-linux32.tar.gz -C release-$(VERSION) .
	mv prox-$(VERSION)-*.tar.gz releases

	# Linux arm
	GOOS=linux GOARCH=arm go build -ldflags "-s -w -X main.Version=$(VERSION)" -o release-$(VERSION)/prox ./cmd/prox
	tar -czf prox-$(VERSION)-linux-arm.tar.gz -C release-$(VERSION) .
	mv prox-$(VERSION)-*.tar.gz releases

	# Mac
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o release-$(VERSION)/prox ./cmd/prox
	tar -czf prox-$(VERSION)-osx.tar.gz -C release-$(VERSION) .
	mv prox-$(VERSION)-*.tar.gz releases

	rm -R release-$(VERSION)


LICENSE-THIRD-PARTY: $(shell find vendor -name LICENSE)
	@echo -e "Third party libraries\n" > $@
	@for f in $$(find vendor -name LICENSE -printf '%P\n' | xargs dirname); do \
		echo "Including license of $$f"; \
		echo "================================================================================" >> $@; \
		echo "$$f" >> $@; \
		echo "================================================================================" >> $@; \
		cat "vendor/$$f/LICENSE" >> $@; \
		echo -e "\n" >> $@; \
	done
