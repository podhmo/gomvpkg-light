default:
	go install -v .
	gomvpkg-light --from github.com/podhmo/handwriting/multifile --to github.com/podhmo/handwriting/multifile2 --in github.com/podhmo/handwriting
