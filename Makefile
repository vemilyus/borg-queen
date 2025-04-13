export CGO_ENABLED = 0


clean:
	rm -rf ./bin


test:
	go test ./credentials/...


test-ci:
	go test -json ./credentials/... \
		| go-ctrf-json-reporter -output ctrf-report.json


build: clean
	go build -o="./bin/cred" ./credentials/cmd/cli
	go build -o="./bin/credstore" ./credentials/cmd/store


build-ci:
	go build -o="./bin/cred-$(SUFFIX)" -ldflags="-s -w -X main.version=$(VERSION)" ./credentials/cmd/cli
	go build -o="./bin/credstore-$(SUFFIX)" -ldflags="-s -w -X main.version=$(VERSION)" ./credentials/cmd/store

	upx -9 ./bin/* || echo ""
