# go simple proxy

    可以建立简单的隧道代理。所有的代理可以放到文件或者redis set中，使用时随机取一个。
    可以直接使用已经构建好的releases/proxy(mac m1)、proxy_linux_amd64、proxy_linux_arm64

    注意：文件或者redis set只能选其一

### dev

    ```
    go run ./cmd/proxy/*.go
    ```

### build

    ```
    make proxy
    ```

### docker

    redis
    ```shell
    docker run -d --name redis -p 6377:6379 redis --requirepass "password"
    ```
    ```
    redis-cli -p 6377 -a password
    ```

### use

    * -s 设置是否server，默认false（client）
    * -v 调试模式（显示文件行数）
    * -q 安静模式（无输出）
    * -ph set proxy host，default "127.0.0.1:8081"
    * -pu set proxy user，default ""
    * -pp set proxy password，default ""
    * -pf set proxy file，default ""
    * -rh set redis host，default "127.0.0.1:6379"
    * -rp set redis password，default ""
    * -rd set redis db，default 0
    * -rk set redis key，default ""

    client
    ```
    go run ./cmd/proxy/*.go

    # add into redis
    ./releases/proxy -pu "user5" -pp "password5" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    ```

    server
    ```
    # read from file
    go run ./cmd/proxy/*.go -s -pu "user1" -pp "password1" -pf ".proxies"
    ./proxy_linux_amd64 -s -ph ":6379" -pu "hMN8yITtYYMx1cSL" -pp "LgpPI4qPDh6xe2Pt" -pf ".proxies"

    # read from redis
    ./releases/proxy -v -s -pu "user" -pp "password" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    ```

### test

    ```
    curl -x 127.0.0.1:8080 https://cip.cc

    # with auth
    curl -x user1:password1@127.0.0.1:8081 http://cip.cc
    ```

### TODO

    * https proxy
    * check same host proxy
    * only server read proxy file
    * docker support

### 赞赏

![image](./appreciate.jpeg)