#!/bin/sh
echo "compiling.."
go build ./cmd/wed

bindir=${GOBIN-${GOPATH-"/usr"}"/bin"}
echo "copying wed to "$bindir
sudo cp wed $bindir
rm wed

if [ -d /usr/share/man/man1/ ]; then
	echo "installing wed manual.."
	gzip -c wed.1 > wed.1.gz
	sudo cp wed.1.gz /usr/share/man/man1/
	rm wed.1.gz
fi

echo "done"
