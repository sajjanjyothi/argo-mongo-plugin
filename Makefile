BINARY_NAME=mongodb-plugin

build:
	go build -o bin/$(BINARY_NAME) ./...

build-docker:
	docker build -t $(BINARY_NAME):latest .