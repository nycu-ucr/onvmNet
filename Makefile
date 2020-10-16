.PHONY: all


all: main.go
	CGO_LDFLAGS_ALLOW="-Wl,(--whole-archive|--no-whole-archive)" go build ./main.go
	mkdir -p build/app
	cp main build/app/goConn
