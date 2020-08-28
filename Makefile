default: build

build:
	go build .

test:
	go test -v $(TEST_ARGS_DEF) $(TEST_ARGS) ./libvirt

vet:
	go vet .*


.PHONY: build test vet
