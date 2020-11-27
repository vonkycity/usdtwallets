all:
	export GO111MODULE=on
	export GOPROXY=https://goproxy.io
	test -d build/Release || mkdir -p build/Release
	go build -o build/Release/main
	test -e build/Release/config.yaml || cp config.yaml build/Release/
	#test -e build/Release/static || mkdir -p build/Release/static
clean:
	rm -rf build/Release/*

linux:
	export GO111MODULE=on
	GOOS=linux GOARCH=amd64 go build -o build/Release/main_linux