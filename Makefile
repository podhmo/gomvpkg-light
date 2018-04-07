PKG = github.com/podhmo/handwriting

default:
	go install -v .
	gomvpkg-light --from $(PKG)/generator/deriving --to $(PKG)/deriving2 --in $(PKG)
