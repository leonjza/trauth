# ref: https://vic.demuzere.be/articles/golang-makefile-crosscompile/

LD_FLAGS := -ldflags="-s -w"
BIN_DIR := build

default: clean darwin linux windows integrity

clean:
	$(RM) $(BIN_DIR)/trauth*
	go clean -x

install:
	go install

darwin:
	GOOS=darwin GOARCH=amd64 go build $(LD_FLAGS) -o '$(BIN_DIR)/trauth-darwin-amd64'

linux:
	GOOS=linux GOARCH=amd64 go build $(LD_FLAGS) -o '$(BIN_DIR)/trauth-linux-amd64'

windows:
	GOOS=windows GOARCH=amd64 go build $(LD_FLAGS) -o '$(BIN_DIR)/trauth-windows-amd64.exe'

integrity:
	cd $(BIN_DIR) && shasum *
