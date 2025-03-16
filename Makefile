SRC = $(shell find . -name '*.go')

rd: $(SRC)
	go mod tidy
	go build .
	go install github.com/hillside-labs/rd

rd-server: $(SRC)
	go mod tidy
	go build ./cli/rd-server
	

