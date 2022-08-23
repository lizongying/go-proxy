.PHONY: all proxy proxy-linux-amd64 proxy-linux-arm64

all: proxy proxy-linux-amd64 proxy-linux-arm64

proxy:
	go vet ./cmd/proxy
	go build -ldflags "-X main.buildTime=`date +%Y%m%d.%H:%M:%S` -X main.buildCommit=`git rev-parse --short=12 HEAD` -X main.buildBranch=`git branch --show-current`" -o ./releases/proxy ./cmd/proxy

proxy-linux-amd64:
	go vet ./cmd/proxy
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.buildTime=`date +%Y%m%d.%H:%M:%S` -X main.buildCommit=`git rev-parse --short=12 HEAD` -X main.buildBranch=`git branch --show-current`" -o ./releases/proxy_linux_amd64 ./cmd/proxy

proxy-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-X main.buildTime=`date +%Y%m%d.%H:%M:%S` -X main.buildCommit=`git rev-parse --short=12 HEAD` -X main.buildBranch=`git branch --show-current`" -o ./releases/proxy_linux_arm64 ./cmd/proxy
