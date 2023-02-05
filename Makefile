all:: bin/api

bin/api:: go.mod *.go */*.go
	mkdir -p bin
	go build -ldflags "-X main.Version=`git describe --tag`" -o bin/api github.com/boynton/api

install:: all
	rm -f $(HOME)/bin/api
	cp -p bin/api $(HOME)/bin/api

test::
#	go test github.com/boynton/api/test

proper::
	go fmt github.com/boynton/api
	go vet github.com/boynton/api
	go fmt github.com/boynton/api/sadl
	go vet github.com/boynton/api/sadl
	go fmt github.com/boynton/api/openapi
	go vet github.com/boynton/api/openapi

clean::
	rm -rf bin

go.mod:
	go mod init github.com/boynton/api && go mod tidy
