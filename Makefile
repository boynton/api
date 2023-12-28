-include _test.make

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
	gofmt -s -w main.go assembly.go common model
	go fmt github.com/boynton/api/golang
	go vet github.com/boynton/api/golang
	gofmt -s -w golang
	go fmt github.com/boynton/api/smithy
	go vet github.com/boynton/api/smithy
	gofmt -s -w smithy
	go fmt github.com/boynton/api/sadl
	go vet github.com/boynton/api/sadl
	gofmt -s -w sadl
	go fmt github.com/boynton/api/openapi
	go vet github.com/boynton/api/openapi
	gofmt -s -w openapi

clean::
	rm -rf bin

go.mod:
	go mod init github.com/boynton/api && go mod tidy
