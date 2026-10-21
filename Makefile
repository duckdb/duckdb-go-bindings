BROOD=GOWORK=off go run ./internal/cmd/brood

fetch.static.libs:
	$(BROOD) fetch

fetch.dynamic.lib:
	$(BROOD) fetch-shared

# Stage a custom build, install its header and select it through go.work.
use.libs:
	$(BROOD) use

# Restore the pinned header and stop selecting the staged native module.
unuse.libs:
	git restore -- include/duckdb.h
	rm -f go.work go.work.sum

update.libs:
	$(BROOD) package
	$(BROOD) pin
	./scripts/test.sh packaged

publish.libs:
	$(BROOD) upload

test:
	./scripts/test.sh

test.static:
	./scripts/test.sh static

test.dynamic:
	./scripts/test.sh dynamic

.PHONY: fetch.static.libs fetch.dynamic.lib use.libs unuse.libs update.libs publish.libs test test.static test.dynamic
