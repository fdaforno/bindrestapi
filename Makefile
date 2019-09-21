# Replace demo with your desired executable name
appname := bind9rest
ver :=$(shell git log -1 --pretty=format:"%H")
date :=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
sources := $(wildcard *.go)

build = GOOS=$(1) GOARCH=$(2) go build -ldflags "-X main.GitHash=$(ver) -X main.BuildStamp=$(date)" -o compiled/$(appname)$(3)
tar = cd compiled && cp -f ../config.toml . && tar -cvzf $(1)_$(2).tar.gz $(appname)$(3) && rm $(appname)$(3)
zip = cd compiled && zip $(1)_$(2).zip $(appname)$(3) && rm $(appname)$(3)

.PHONY: all windows darwin linux clean

all: osx linux
#all: windows darwin linux

clean:
    rm -rf compiled/

##### LINUX BUILDS #####
linux: compiled/linux_arm.tar.gz compiled/linux_arm64.tar.gz compiled/linux_386.tar.gz compiled/linux_amd64.tar.gz

compiled/linux_386.tar.gz: $(sources)
    $(call build,linux,386,)
    $(call tar,linux,386)

compiled/linux_amd64.tar.gz: $(sources)
    $(call build,linux,amd64,)
    $(call tar,linux,amd64)

compiled/linux_arm.tar.gz: $(sources)
    $(call build,linux,arm,)
    $(call tar,linux,arm)

compiled/linux_arm64.tar.gz: $(sources)
    $(call build,linux,arm64,)
    $(call tar,linux,arm64)

##### DARWIN (MAC) BUILDS #####
osx: compiled/osx.tar.gz

compiled/osx.tar.gz: $(sources)
    $(call build,darwin,amd64,)
    $(call tar,darwin,amd64)

##### WINDOWS BUILDS #####
#windows: compiled/windows_386.zip build/windows_amd64.zip

#compiled/windows_386.zip: $(sources)
#   $(call build,windows,386,.exe)
#   $(call zip,windows,386,.exe)

#compiled/windows_amd64.zip: $(sources)
#   $(call build,windows,amd64,.exe)
#   $(call zip,windows,amd64,.exe)
