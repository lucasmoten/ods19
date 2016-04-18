
#!/bin/bash

set -e -x

echo "Creating release dir..."
mkdir -p release

# variables as defined by "go tool nm"
OSVAR=decipher.com/object-drive-server/cryptotest.BuildOS
ARCHVAR=decipher.com/object-drive-server/cryptotest.BuildARCH
ARMVAR=decipher.com/object-drive-server/cryptotest.BuildARM

createRelease() {
	os=$1
	arch=$2
	arm=$3

	if [ "$os" = darwin ]
	then
		osname='mac'
	else
		osname=$os
	fi
	if [ "$arch" = amd64 ]
	then
		osarch=64bit
	else
		osarch=32bit
	fi

	ldflags="-X $OSVAR $os -X $ARCHVAR $arch"
	if [ "$arm" ]
	then
		osarch=arm-v$arm
		ldflags="$ldflags -X $ARMVAR $arm"
	fi

	binname=cryptotest
	if [ "$osname" = windows ]
	then
		binname="$binname.exe"
	fi

	relname="../release/odrive-$osname-$osarch"
	echo "Creating $os/$arch binary..."

	if [ "$arm" ]
	then
		GOOS=$os GOARCH=$arch GOARM=$arm go build -ldflags "$ldflags" -o "out/$binname" cmd/cryptotest/cryptotest.go
	else
		GOOS=$os GOARCH=$arch go build -ldflags "$ldflags" -o "out/$binname" cmd/cryptotest/cryptotest.go
	fi

	cd out

	if [ "$osname" = windows ]
	then
		zip "$relname.zip" "$binname"
	else
		tar cvzf "$relname.tgz" "$binname"
	fi
	cd ..
}

# Mac Releases
createRelease darwin 386
createRelease darwin amd64

# Linux Releases
createRelease linux 386
createRelease linux amd64

# ARM Releases
createRelease linux arm 5
createRelease linux arm 6
createRelease linux arm 7

# Windows Releases
createRelease windows 386
createRelease windows amd64
