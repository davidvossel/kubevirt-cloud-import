#!/bin/bash

which golint
if [ $? -ne 0 ]; then
	echo "Downloading golint tool"
	go get -u golang.org/x/lint/golint
fi

RETVAL=0
# ignore the vendor folder here if it exists
for file in $(find . -path ./vendor -prune -o -type f -name '*.go' -print); do
	golint -set_exit_status "$file"
	if [[ $? -ne 0 ]]; then 
		RETVAL=1
 	fi
done
exit $RETVAL
