.PHONY: all build clean test

all: clean build

TARGET := bind9rest
.DEFAULT_GOAL: $(TARGET)

DATE := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
BUILD := $(shell git log -1 --pretty=format:"%H")

# Use linker flags to provide version/build settings to the target
LDFLAGS=-ldflags "-X=main.BuildStamp=$(DATE) -X=main.GitHash=$(BUILD)"

SRC = $(wildcard *.go)

$(TARGET): $(SRC)
	@go build $(LDFLAGS) -o $(TARGET)

test:
	@go test -v -race $(SRC)

build: $(TARGET)
	@true

clean:
	@go clean -i $(SRC)
	@rm -f $(TARGET)