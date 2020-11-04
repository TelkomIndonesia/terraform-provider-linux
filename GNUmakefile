
default: build

build: 
	go build

test:
	go test ./... $(TESTARGS)

testacc:
	TF_ACC=1 go test ./... $(TESTARGS) -timeout 120m

.PHONY: build test testacc