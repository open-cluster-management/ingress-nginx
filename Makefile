include build/Configfile

-include $(shell curl -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/open-cluster-management/build-harness-extensions/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)

.PHONY: build doc fmt lint run test vendor_clean vendor_get vendor_update vet

deps:
	go get golang.org/x/lint/golint
	go get -u github.com/apg/patter
	go get -u github.com/wadey/gocovmerge
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

lint:
	golint -set_exit_status=true pkg/
	golint -set_exit_status=true cmd/

build:
	go build -v -i -o bin/management-ingress github.com/open-cluster-management/management-ingress/cmd/nginx

docker-binary:
	CGO_ENABLED=0 go build -a -installsuffix cgo -v -i -o rootfs/management-ingress github.com/open-cluster-management/management-ingress/cmd/nginx
	strip rootfs/management-ingress

test:
	@./build/test.sh

coverage:
	go tool cover -html=cover.out -o=cover.html
	@./build/calculate-coverage.sh

fmt:
	gofmt -l ${GOFILES}

vet:
	gometalinter  --deadline=1000s --disable-all --enable=vet --enable=vetshadow --enable=ineffassign --enable=goconst --tests  --vendor ./...
