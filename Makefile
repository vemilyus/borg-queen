export CGO_ENABLED = 0


clean:
	rm -rf ./bin


test:
	go test ./credentials/...


build: clean
	go build -o="./bin/cred" ./credentials/cmd/cli
	go build -o="./bin/credstore" ./credentials/cmd/store


build-ci:
	go build -o="./bin/cred-$(SUFFIX)" -ldflags="-s -w -X main.version=$(VERSION)" ./credentials/cmd/cli
	go build -o="./bin/credstore-$(SUFFIX)" -ldflags="-s -w -X main.version=$(VERSION)" ./credentials/cmd/store

	upx -9 ./bin/* || echo ""
