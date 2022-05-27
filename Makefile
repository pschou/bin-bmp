PROG_NAME := "bin-bmp"
VERSION = 0.1.$(shell date +%Y%m%d.%H%M)
FLAGS := "-s -w -X main.version=${VERSION}"


build:
	CGO_ENABLED=0 go build -ldflags=${FLAGS} -o ${PROG_NAME} *.go

examples:
	./bin-bmp bin-bmp bin-bmp.bmp
	./bin-bmp /usr/bin/ssh ssh.bmp
	cp bin-bmp bin-bmp-upx
	upx --ultra-brute bin-bmp-upx
	./bin-bmp bin-bmp-upx bin-bmp-upx.bmp
