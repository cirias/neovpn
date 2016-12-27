build:
	cd server/ && go build
	cd client/ && go build

buildlinux:
	cd server/ && GOOS=linux GOARCH=amd64 go build
	cd client/ && GOOS=linux GOARCH=amd64 go build

pack:
	-mkdir -p dist/server
	-mkdir -p dist/client
	cp server/server dist/server/
	cp script/server/*.sh dist/server/
	cp client/client dist/client/
	cp script/client/*.sh dist/client/
