PKG = github.com/podhmo/handwriting

light:
	go install -v .
	cd $(GOPATH)/src/$(PKG); rm -rf deriving2; git reset --hard HEAD
	gomvpkg-light --from $(PKG)/generator/deriving --to $(PKG)/deriving2 --in $(PKG) --disable-gc

unsafe:
	go install -v .
	cd $(GOPATH)/src/$(PKG); rm -rf deriving2; git reset --hard HEAD
	gomvpkg-light --from $(PKG)/generator/deriving --to $(PKG)/deriving2 --in $(PKG) --unsafe

original:
	cd $(GOPATH)/src/$(PKG); rm -rf deriving2; git reset --hard HEAD
	gomvpkg -from $(PKG)/generator/deriving -to $(PKG)/deriving2
