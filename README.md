# go simple proxy

    可以建立简单的隧道代理。所有的代理可以放到redis set中，使用时随机取一个。

### dev

    ```
    go run main.go
    ```

### build

    linux
    ```
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o go-proxy-linux
    ```
    mac
    ```
    go build -o go-proxy-mac
    ```

### use

    * -h 设置公开的host，默认":8081"
    * -pu 设置代理的用户名，默认""
    * -pp 设置代理的密码，默认""
    * -rh 设置redis的host，默认"127.0.0.1:6379"
    * -rp 设置redis的password，默认""
    * -rd 设置redis的db，默认0
    * -rk 设置redis的key，默认"proxies"

    ```
    ./go-proxy-mac -h ":8081" -pu "" -pp "" -rh "127.0.0.1:6379" -rp "" -rd 0 -rk "proxies"
    ```

### test

    ```
    curl -x 127.0.0.1:8081 https://cip.cc
    curl -x 127.0.0.1:8081 http://cip.cc
    ```
