build:
	make clean
	make build-rando
	make build-randoui
	make build-docker

clean:
	rm -rf ./bin
	rm -rf ./dist

provision:
	mkdir -p ./bin
	go build -o ./bin/provision ./cmd/provision
	./cmd/provision/bin/provision-davebot.sh

build-rando:
	mkdir -p ./dist/bin
	GOOS=linux GOARCH=amd64 go build -o ./dist/bin/rando ./cmd/rando

build-randoui:
	mkdir -p ./dist/bin
	GOOS=linux GOARCH=amd64 go build -o ./dist/bin/randoui ./cmd/randoui

build-docker:
	docker build .
