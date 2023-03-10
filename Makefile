.PHONY: test install release version

VERSION=$(shell git describe --dirty)

test:
	go test -race -cover -mod=readonly

install:
	go build -ldflags "-s -w -X main.Version=$(VERSION)" -o $$GOPATH/bin/prox ./cmd/prox

release: test LICENSE-THIRD-PARTY
	goreleaser release --clean

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

