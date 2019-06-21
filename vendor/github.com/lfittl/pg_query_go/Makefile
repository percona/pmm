.PHONY: default build test benchmark update_source clean

default: test

build:
	go build

test: build
	go get github.com/kr/pretty
	go test -v ./ ./nodes

benchmark:
	go build -a
	go test -test.bench=. -test.run=XXX -test.benchtime 10s -test.benchmem -test.cpu=4
	#go test -c -o benchmark
	#GODEBUG=schedtrace=100 ./benchmark -test.bench=BenchmarkRawParseCreateTableParallel -test.run=XXX -test.benchtime 20s -test.benchmem -test.cpu=16

# --- Below only needed for releasing new versions

LIB_PG_QUERY_TAG = 10-1.0.2

root_dir := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
LIB_TMPDIR = $(root_dir)/tmp
LIBDIR = $(LIB_TMPDIR)/libpg_query
LIBDIRGZ = $(TMPDIR)/libpg_query-$(LIB_PG_QUERY_TAG).tar.gz

$(LIBDIR): $(LIBDIRGZ)
	mkdir -p $(LIBDIR)
	cd $(LIB_TMPDIR); tar -xzf $(LIBDIRGZ) -C $(LIBDIR) --strip-components=1

$(LIBDIRGZ):
	mkdir -p $(LIB_TMPDIR)
	curl -o $(LIBDIRGZ) https://codeload.github.com/lfittl/libpg_query/tar.gz/$(LIB_PG_QUERY_TAG)

update_source: $(LIBDIR)
	rm -f parser/*.{c,h}
	rm -fr parser/include
	# Reduce everything down to one directory
	cp -a $(LIBDIR)/src/* parser/
	mv parser/postgres/* parser/
	rmdir parser/postgres
	cp -a $(LIBDIR)/pg_query.h parser/include
	# Make sure every .c file in the top-level directory is its own translation unit
	mv parser/*{_conds,_defs,_helper}.c parser/include
	# Other support files
	rm -fr testdata
	cp -a $(LIBDIR)/testdata testdata
	# Update nodes directory
	ruby scripts/generate_nodes.rb

clean:
	-@ $(RM) -r $(LIB_TMPDIR)
