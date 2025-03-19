split.artefacts:
	mkdir ${DIRECTORY}/libs/core && \
	mkdir ${DIRECTORY}/libs/corefunctions && \
	mkdir ${DIRECTORY}/libs/parquet && \
	mkdir ${DIRECTORY}/libs/icu && \
	mv ${DIRECTORY}/libs/libduckdb.a ${DIRECTORY}/libs/core/libduckdb.a && \
	cd ${DIRECTORY}/libs/core && \
	${AR} -x libduckdb.a && \
	rm -f __.SYMDEF && \
	rm -f libduckdb.a && \
	mv ub_duckdb_core_functions_algebraic.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_distributive.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_nested.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_regression.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_array.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_bit.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_blob.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_date.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_debug.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_enum.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_generic.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_list.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_map.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_math.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_operators.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_random.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_string.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_struct.cpp.${FILETYPE} ../corefunctions/ && \
	mv ub_duckdb_core_functions_union.cpp.${FILETYPE} ../corefunctions/ && \
	mv lambda_functions.cpp.${FILETYPE} ../corefunctions/ && \
	mv column_reader.cpp.${FILETYPE} ../parquet/ && \
	mv column_writer.cpp.${FILETYPE} ../parquet/ && \
	mv parquet_crypto.cpp.${FILETYPE} ../parquet/ && \
	mv parquet_extension.cpp.${FILETYPE} ../parquet/ && \
	mv parquet_metadata.cpp.${FILETYPE} ../parquet/ && \
	mv parquet_reader.cpp.${FILETYPE} ../parquet/ && \
	mv parquet_statistics.cpp.${FILETYPE} ../parquet/ && \
	mv parquet_timestamp.cpp.${FILETYPE} ../parquet/ && \
	mv parquet_writer.cpp.${FILETYPE} ../parquet/ && \
	mv serialize_parquet.cpp.${FILETYPE} ../parquet/ && \
	mv zstd_file_system.cpp.${FILETYPE} ../parquet/ && \
	mv geo_parquet.cpp.${FILETYPE} ../parquet/ && \
	mv parquet_types.cpp.${FILETYPE} ../parquet/ && \
	mv TProtocol.cpp.${FILETYPE} ../parquet/ && \
	mv TTransportException.cpp.${FILETYPE} ../parquet/ && \
	mv TBufferTransports.cpp.${FILETYPE} ../parquet/ && \
	mv snappy.cc.${FILETYPE} ../parquet/ && \
	mv snappy-sinksource.cc.${FILETYPE} ../parquet/ && \
	mv lz4.cpp.${FILETYPE} ../parquet/ && \
	mv dictionary_hash.cpp.${FILETYPE} ../parquet/ && \
	mv backward_references_hq.cpp.${FILETYPE} ../parquet/ && \
	mv histogram.cpp.${FILETYPE} ../parquet/ && \
	mv memory.cpp.${FILETYPE} ../parquet/ && \
	mv entropy_encode.cpp.${FILETYPE} ../parquet/ && \
	mv compound_dictionary.cpp.${FILETYPE} ../parquet/ && \
	mv compress_fragment_two_pass.cpp.${FILETYPE} ../parquet/ && \
	mv block_splitter.cpp.${FILETYPE} ../parquet/ && \
	mv command.cpp.${FILETYPE} ../parquet/ && \
	mv encode.cpp.${FILETYPE} ../parquet/ && \
	mv encoder_dict.cpp.${FILETYPE} ../parquet/ && \
	mv cluster.cpp.${FILETYPE} ../parquet/ && \
	mv backward_references.cpp.${FILETYPE} ../parquet/ && \
	mv utf8_util.cpp.${FILETYPE} ../parquet/ && \
	mv compress_fragment.cpp.${FILETYPE} ../parquet/ && \
	mv fast_log.cpp.${FILETYPE} ../parquet/ && \
	mv brotli_bit_stream.cpp.${FILETYPE} ../parquet/ && \
	mv bit_cost.cpp.${FILETYPE} ../parquet/ && \
	mv static_dict.cpp.${FILETYPE} ../parquet/ && \
	mv literal_cost.cpp.${FILETYPE} ../parquet/ && \
	mv metablock.cpp.${FILETYPE} ../parquet/ && \
	mv dictionary.cpp.${FILETYPE} ../parquet/ && \
	mv constants.cpp.${FILETYPE} ../parquet/ && \
	mv transform.cpp.${FILETYPE} ../parquet/ && \
	mv platform.cpp.${FILETYPE} ../parquet/ && \
	mv shared_dictionary.cpp.${FILETYPE} ../parquet/ && \
	mv context.cpp.${FILETYPE} ../parquet/ && \
	mv state.cpp.${FILETYPE} ../parquet/ && \
	mv decode.cpp.${FILETYPE} ../parquet/ && \
	mv huffman.cpp.${FILETYPE} ../parquet/ && \
	mv bit_reader.cpp.${FILETYPE} ../parquet/ && \
	mv ub_duckdb_icu_common.cpp.${FILETYPE} ../icu/ && \
	mv ub_duckdb_icu_i18n.cpp.${FILETYPE} ../icu/ && \
	mv stubdata.cpp.${FILETYPE} ../icu/ && \
	mv icu_extension.cpp.${FILETYPE} ../icu/ && \
	mv icu-current.cpp.${FILETYPE} ../icu/ && \
	mv icu-dateadd.cpp.${FILETYPE} ../icu/ && \
	mv icu-datefunc.cpp.${FILETYPE} ../icu/ && \
	mv icu-datepart.cpp.${FILETYPE} ../icu/ && \
	mv icu-datesub.cpp.${FILETYPE} ../icu/ && \
	mv icu-datetrunc.cpp.${FILETYPE} ../icu/ && \
	mv icu-makedate.cpp.${FILETYPE} ../icu/ && \
	mv icu-list-range.cpp.${FILETYPE} ../icu/ && \
	mv icu-table-range.cpp.${FILETYPE} ../icu/ && \
	mv icu-strptime.cpp.${FILETYPE} ../icu/ && \
	mv icu-timebucket.cpp.${FILETYPE} ../icu/ && \
	mv icu-timezone.cpp.${FILETYPE} ../icu/ && \
	cd .. && \
	${AR} r libcore.a core/* && \
	${AR} r libcorefunctions.a corefunctions/* && \
	${AR} r libparquet.a parquet/* && \
	${AR} r libicu.a icu/* && \
	rm -rf core && \
	rm -rf corefunctions && \
	rm -rf parquet && \
	rm -rf icu

fetch.static.lib:
	cd ${DIRECTORY} && \
	curl -OL https://github.com/duckdb/duckdb/releases/download/${VERSION}/${FILENAME}.zip && \
	rm -rf libs && \
	mkdir libs && \
	rm -f duckdb.h && \
	unzip ${FILENAME}.zip && \
	mv libduckdb_bundle.a libs/libduckdb.a && \
	rm -f ${FILENAME}.zip

fetch.static.lib.mingw:
	cd ${DIRECTORY} && \
	curl -OL https://github.com/duckdb/duckdb/releases/download/v1.2.1/static-lib-windows-mingw.zip && \
	rm -rf libs && \
	mkdir libs && \
	rm -f duckdb.h && \
	unzip static-lib-windows-mingw.zip && \
	mv libduckdb_bundle.a libs/libduckdb.a && \
	rm -f static-lib-windows-mingw.zip

update.binding:
	rm -f ${DIRECTORY}/bindings.go && \
	cp bindings.go ${DIRECTORY}/bindings.go

test.dynamic.lib:
	mkdir dynamic-dir && \
	cd dynamic-dir && \
	curl -OL https://github.com/duckdb/duckdb/releases/download/v1.2.0/${FILENAME}.zip && \
	unzip ${FILENAME}.zip
