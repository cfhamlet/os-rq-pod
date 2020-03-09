#!/usr/bin/env bash

covermode=${COVERMODE:-atomic}
coverdir=$(mktemp -d /tmp/coverage.XXXXXXXXXX)
profile="${coverdir}/cover.out"

echo "profile: ${profile}"

push_to_codecov() {
    cp ${profile} coverage.txt
    bash <(curl -s https://codecov.io/bash)
    rm coverage.txt
}

go test -race -coverprofile=${profile} -covermode="${covermode}" ./...
go tool cover -func ${profile}

case "${1-}" in
  --html)
    go tool cover -html "${profile}"
    ;;
  --codecov)
    push_to_codecov
    ;;
esac
