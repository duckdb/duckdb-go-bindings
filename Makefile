DUCKDB_VERSION=v2.0.0-alpha46057

# Where artifacts are fetched from. Override on the make command line to install
# from somewhere else, e.g. the nightly workflow's staging endpoint. It must be
# passed as an argument, not exported: '=' takes precedence over the environment.
# TODO: Fetch from the GitHub release once DuckDB v2.0.0 is out.
BASE_URL=https://duckdb-staging.duckdb.org/b15358c371/${DUCKDB_VERSION}/duckdb/duckdb/github_release

# Where fetch.dynamic.lib unpacks the shared library.
DYNAMIC_DIR=dynamic-dir

# What fetch.static.libs keeps from the static libs tarball: the archives the
# prebuilt platform packages link, and the header those packages compile against.
# The tarball also carries the CLI's libduckdb_shell.a, the C API v2 and
# extension headers, and on some platforms additional extensions.
STATIC_LIBS+=libautocomplete_extension.a
STATIC_LIBS+=libcore_functions_extension.a
STATIC_LIBS+=libduckdb_httplib.a
STATIC_LIBS+=libduckdb_loadable_extensions.a
STATIC_LIBS+=libduckdb_static.a
STATIC_LIBS+=libicu_extension.a
STATIC_LIBS+=libjson_extension.a
STATIC_LIBS+=libparquet_extension.a
STATIC_LIBS+=libtpcds_extension.a
STATIC_LIBS+=libtpch_extension.a

fetch.static.libs: FILENAME=duckdb-static-libs-${ARTIFACT_PLATFORM}.tar.gz
fetch.static.libs:
	cd lib/${PLATFORM} && \
	curl --fail --retry 5 -OL ${BASE_URL}/${FILENAME} && \
	rm -f *.a duckdb.h && \
	tar -xzf ${FILENAME} ${STATIC_LIBS} duckdb.h && \
	if [ -n "${COPY_HEADER}" ]; then \
		tar -xzf ${FILENAME} -C ../../include duckdb.h duckdb_static_extension.h; \
	fi && \
	rm -f ${FILENAME}

fetch.dynamic.lib: FILENAME=duckdb-shared-libs-${ARTIFACT_PLATFORM}.tar.gz
fetch.dynamic.lib:
	mkdir -p ${DYNAMIC_DIR} && \
	cd ${DYNAMIC_DIR} && \
	curl --fail --retry 5 -OL ${BASE_URL}/${FILENAME} && \
	tar -xzf ${FILENAME} && \
	rm -f ${FILENAME}

# Packaging is separate so nightly tests still exercise the upstream artifacts.
# LLVM tools are needed only when an archive exceeds GitHub's 100 MiB limit.
package.static.libs:
	python3 scripts/package_static_libs.py lib/${PLATFORM}

.PHONY: fetch.static.libs fetch.dynamic.lib package.static.libs
