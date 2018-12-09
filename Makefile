.PHONY: test install release version

VERSION=$(shell git describe --dirty)

test:
	go test -race -cover -mod=readonly

install:
	go build -ldflags "-s -w -X main.Version=$(VERSION)" -o $$GOPATH/bin/prox ./cmd/prox

release: test LICENSE-THIRD-PARTY
	mkdir -p releases
	mkdir -p release-$(VERSION)
	cp LICENSE release-$(VERSION)
	cp LICENSE-THIRD-PARTY release-$(VERSION)
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

.PHONY: LICENSE-THIRD-PARTY
LICENSE-THIRD-PARTY:
	@echo -e "Third party libraries\n" > $@
	for dependency in $$(go list -m -f '{{ .Dir }}' all | grep -v prox); do \
		license=$$(find "$$dependency" -name LICENSE -o -name COPYING | tail -n1); \
		if [[ -n "$$license" ]]; then \
			name=$$(echo "$$dependency" | sed "s;$$GOPATH/pkg/mod/;;"); \
			echo "Including license of $$name"; \
			echo "================================================================================" >> $@; \
			echo "$$name" >> $@; \
			echo "================================================================================" >> $@; \
			cat "$$license" >> $@; \
			echo -e "\n" >> $@; \
		fi; \
	done

version:
	@echo $(VERSION)
