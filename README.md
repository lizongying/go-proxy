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
    * -ph 设置公开的host，默认":8081"
    * -pu 设置代理的用户名，默认""
    * -pp 设置代理的密码，默认""
    * -pf 设置代理的列表文件，默认""
    * -rh 设置redis的host，默认"127.0.0.1:6379"
    * -rp 设置redis的password，默认""
    * -rd 设置redis的db，默认0
    * -rk 设置redis的key，默认""

    client
    ```
    go run ./cmd/proxy/*.go -ph "127.0.0.1:8081"
    ./proxy_linux_amd64 -ph "127.0.0.1:8081" -pu "" -pp ""

    go run ./cmd/proxy/*.go -ph "127.0.0.1:8081" -pu "user1" -pp "password1" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    go run ./cmd/proxy/*.go -ph "127.0.0.1:8082" -pu "user2" -pp "password2" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    go run ./cmd/proxy/*.go -q -ph "127.0.0.1:8083" -pu "" -pp "" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    go run ./cmd/proxy/*.go -v -ph "127.0.0.1:8084" -pu "" -pp "" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    ./releases/proxy -ph "127.0.0.1:8085" -pu "user5" -pp "password5" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    ```

    server
    ```
    go run ./cmd/proxy/*.go -s -ph "127.0.0.1:8081" -pu "user1" -pp "password1" -pf ".proxies"
    go run ./cmd/proxy/*.go -v -s -ph "0.0.0.0:8090" -pu "user" -pp "password" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    ./releases/proxy -v -s -ph "0.0.0.0:8090" -pu "user" -pp "password" -rh "127.0.0.1:6377" -rp "password" -rd 0 -rk "proxies"
    ```

### test

    ```
    curl -x 127.0.0.1:8081 https://cip.cc
    
    curl -x 127.0.0.1:8080 https://cip.cc
    curl -x user1:password1@127.0.0.1:8081 http://cip.cc
    curl -x user2:password2@127.0.0.1:8082 http://cip.cc
    curl -x 127.0.0.1:8083 http://cip.cc
    curl -x 127.0.0.1:8084 http://cip.cc
    curl -x user:password@127.0.0.1:8090 http://cip.cc
    ```

### TODO

    * https proxy
    * check same host proxy
    * only server read proxy file

### 赞赏

![image](./appreciate.jpeg)