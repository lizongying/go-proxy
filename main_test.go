package main_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/go-redis/redis"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
)

// 代理
const proxy = true
const proxyAuth = true
const ProxyUser = "databurning"
const ProxyPassword = "2tQJl*t8@{}"

// 转发服务
const proxyServer = ":8081"

var (
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

var rdb *redis.Client

const (
	RedisHost     = "172.24.16.57"
	RedisPort     = "6379"
	RedisPassword = ""
	RedisDb       = 12
)

func init() {
	errFile, err := os.OpenFile("errors.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("打开日志文件失败：", err)
	}

	Info = log.New(os.Stdout, "Info:", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout, "Warning:", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(io.MultiWriter(os.Stderr, errFile), "Error:", log.Ldate|log.Ltime|log.Lshortfile)

	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", RedisHost, RedisPort),
		Password: RedisPassword,
		DB:       RedisDb,
	})
}

func main() {
	l, err := net.Listen("tcp", proxyServer)
	if err != nil {
		log.Panic(err)
	}
	for {
		client, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handle(client)
	}
}
func handle(client net.Conn) {
	if client == nil {
		return
	}

	defer func() {
		err := client.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}()

	var b [1024]byte
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}
	var method, URL, address string
	_, err = fmt.Sscanf(string(b[:bytes.IndexByte(b[:], '\n')]), "%s%s", &method, &URL)
	if err != nil {
		log.Println(err)
		return
	}

	hostPortURL, err := url.Parse(URL)
	if err != nil {
		log.Println(err)
		return
	}
	if method == "CONNECT" {

		//https
		address = hostPortURL.Scheme + ":" + hostPortURL.Opaque
	} else {

		//http
		if strings.Index(hostPortURL.Host, ":") == -1 {

			//host不带端口， 默认80
			address = hostPortURL.Host + ":80"
		} else {
			address = hostPortURL.Host
		}
	}

	r, err := rdb.SRandMember("pycrawler_proxies:dly").Result()
	if err != nil {
		log.Println(err)
		return
	}
	var proxyHost = r
	var proxyUser = ProxyUser
	var proxyPassword = ProxyPassword

	//拨号
	if proxy {
		address = proxyHost
	}
	server, err := net.Dial("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}
	if method == "CONNECT" {
		if proxy {
			setProxyHeader(b, server, proxyUser, proxyPassword)
		} else {
			_, err = fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
		}
		if err != nil {
			log.Println(err)
			return
		}
	} else {
		if proxy {
			setProxyHeader(b, server, proxyUser, proxyPassword)
		} else {
			_, err = server.Write(b[:n])
		}
		if err != nil {
			log.Println(err)
			return
		}
	}

	//转发
	go func() {
		_, err = io.Copy(server, client)
		if err != nil {
			log.Println(err)
			return
		}
		Error.Println("👌")
	}()
	_, err = io.Copy(client, server)
	if err != nil {
		log.Println(err)
		return
	}
}

func setProxyHeader(b [1024]byte, server net.Conn, proxyUser string, proxyPassword string) {
	var proxyAuthorization = []byte(fmt.Sprintf("Proxy-Authorization: Basic %s\r\n\r\n", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxyUser, proxyPassword)))))
	var header []byte
	for _, v := range b {
		if v == 0 {
			break
		}
		header = append(header, v)
	}
	if proxyAuth {
		header = header[:len(header)-2]
		for _, v := range proxyAuthorization {
			header = append(header, v)
		}
	}
	_, _ = server.Write(header)
}
