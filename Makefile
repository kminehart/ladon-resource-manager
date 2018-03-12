update_codegen:
	./hack/update-codegen.sh

build: update_codegen
	go build -o ./bin/ladon-resource-manager .

build_static: # update_codegen
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o ./bin/ladon-resource-manager .

build_docker: build_static
	docker build . -t "kminehart/ladon-resource-manager:latest"
