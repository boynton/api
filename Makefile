all:: bin/api
#	./bin/api -g json _examples/simple-api.json
#	./bin/api -g api _examples/simple-api.json
	./bin/api -g json /tmp/simple.api
#	./bin/api -g markdown /tmp/test.smithy > /tmp/test.md
#	./bin/api -g markdown /tmp/test.json > /tmp/test.md
#	./bin/api -g markdown _examples/crudl.smithy > /tmp/test.md
#	./bin/api -g model _examples/crudl-openapi.json
#	./bin/api -g model _examples/bufferapp-swagger.json
#	mkdir -p _crudl
#	./bin/api -f -o _crudl -g golang -a golang.inlineSlicesAndMaps=true -a golang.timestampPackage=github.com/boynton/data _examples/crudl.smithy
#	./bin/api -f -o _crudl -g golang -a golang.inlinePrimitives=true -a golang.inlineSlicesAndMaps=true -a golang.timestampPackage=github.com/boynton/data _examples/crudl.smithy
#	(cd _crudl/server; go build; ./server)

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
