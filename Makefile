BINDDIR := /usr/bin

build:
	./build.sh
all: build
install:
	mkdir -p ${DESTDIR}${BINDDIR}
	cp build/linux/x86_64/Linux-x86_64-lpmx ${DESTDIR}${BINDDIR}/lpmx

