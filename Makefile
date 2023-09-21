BINARY_NAME=mongodb-plugin

build:
	go build ./... -o bin/$(BINARY_NAME) -v

build-docker:
	docker build -t $(BINARY_NAME):latest .