DUCKDB_VERSION=v1.5.5

# Where artifacts are fetched from. Override on the make command line to install
# from somewhere else, e.g. the nightly workflow's staging endpoint. It must be
# passed as an argument, not exported: '=' takes precedence over the environment.
BASE_URL=https://github.com/duckdb/duckdb/releases/download/${DUCKDB_VERSION}

# Where fetch.dynamic.lib unpacks the shared library.
DYNAMIC_DIR=dynamic-dir

fetch.static.libs:
	cd lib/${PLATFORM} && \
	curl --fail --retry 5 -OL ${BASE_URL}/${FILENAME}.zip && \
	rm -f *.a duckdb.h && \
	unzip -q ${FILENAME}.zip && \
	rm -f ${FILENAME}.zip && \
	if [ -n "${COPY_HEADER}" ]; then cp duckdb.h ../../include/; fi

fetch.dynamic.lib:
	mkdir -p ${DYNAMIC_DIR} && \
	cd ${DYNAMIC_DIR} && \
	curl --fail --retry 5 -OL ${BASE_URL}/${FILENAME}.zip && \
	unzip -o -q ${FILENAME}.zip && \
	rm -f ${FILENAME}.zip

.PHONY: fetch.static.libs fetch.dynamic.lib
