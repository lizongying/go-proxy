package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
	"time"
)

const proxy = true
const proxyAuth = true

var ProxyServer string

var ProxyUser string
var ProxyPassword string

var RedisHost string
var RedisPassword string
var RedisDb int
var RedisKey string

var rdb *redis.Client

func init() {
	proxyServer := flag.String("h", ":8081", "proxy-host")
	proxyUser := flag.String("pu", "", "proxy-user")
	proxyPassword := flag.String("pp", "", "proxy-password")
	redisHost := flag.String("rh", "127.0.0.1:6379", "redis-host")
	redisPassword := flag.String("rp", "", "redis-password")
	redisDb := flag.Int("rd", 0, "redis-db")
	redisKey := flag.String("rk", "proxies", "redis-key")
	flag.Parse()
	ProxyServer = *proxyServer
	ProxyUser = *proxyUser
	ProxyPassword = *proxyPassword
	RedisHost = *redisHost
	RedisPassword = *redisPassword
	RedisDb = *redisDb
	RedisKey = *redisKey
	log.Println(fmt.Sprintf("proxy-server: %s", ProxyServer))
	log.Println(fmt.Sprintf("redis-key: %s", RedisKey))
	rdb = redis.NewClient(&redis.Options{
		Addr:     RedisHost,
		Password: RedisPassword,
		DB:       RedisDb,
	})
}

func main() {
	l, err := net.Listen("tcp", ProxyServer)
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
	if bytes.IndexByte(b[:], '\n') == -1 {
		log.Println("empty")
		return
	}
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

	proxyHost, err := rdb.SRandMember(RedisKey).Result()
	if err != nil {
		log.Println(err)
		return
	}
	var proxyUser = ProxyUser
	var proxyPassword = ProxyPassword

	//拨号
	if proxy {
		address = proxyHost
	}
	dialer := net.Dialer{Timeout: time.Minute}
	server, err := dialer.Dial("tcp", address)
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
