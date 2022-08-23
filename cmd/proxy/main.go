package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	"io"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

var IsServer bool
var IsDebug bool
var IsQuiet bool

var ProxyServer string

var ProxyHost string
var ProxyUser string
var ProxyPassword string
var ProxyFile string

var RedisHost string
var RedisPassword string
var RedisDb int
var RedisKey string

var logger *log.Logger
var rdb *redis.Client

var Proxies []string

var Auth string

func init() {
	proxyHost := flag.String("ph", ":8081", "proxyHost")
	proxyUser := flag.String("pu", "", "proxyUser")
	proxyPassword := flag.String("pp", "", "proxyPassword")
	proxyFile := flag.String("pf", "", "proxyFile")
	redisHost := flag.String("rh", "127.0.0.1:6379", "redisHost")
	redisPassword := flag.String("rp", "", "redisPassword")
	redisDb := flag.Int("rd", 0, "redisDb")
	redisKey := flag.String("rk", "", "redisKey")
	isServer := flag.Bool("s", false, "isServer")
	isDebug := flag.Bool("v", false, "isDebug")
	isQuiet := flag.Bool("q", false, "isQuiet")
	flag.Parse()
	ProxyHost = *proxyHost
	ProxyUser = *proxyUser
	ProxyPassword = *proxyPassword
	ProxyFile = *proxyFile
	RedisHost = *redisHost
	RedisPassword = *redisPassword
	RedisDb = *redisDb
	RedisKey = *redisKey
	IsServer = *isServer
	IsDebug = *isDebug
	IsQuiet = *isQuiet
	if IsDebug {
		logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Llongfile)
	} else {
		logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	}
	if IsQuiet {
		logger.SetOutput(io.Discard)
	}
	if ProxyUser != "" && ProxyPassword != "" {
		Auth = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", ProxyUser, ProxyPassword)))
	}

	if IsServer {
		logger.Println("mode: server")
	} else {
		logger.Println("mode: client")
		ProxyServer = ProxyHost
		if Auth != "" {
			ProxyServer = fmt.Sprintf("%s:%s@%s", ProxyUser, ProxyPassword, ProxyHost)
		}
	}

	if Auth != "" {
		logger.Println("proxy:", fmt.Sprintf("%s:%s@%s", ProxyUser, ProxyPassword, ProxyHost))
	} else {
		logger.Println("proxy:", ProxyHost)
	}

	if ProxyFile != "" {
		logger.Println(fmt.Sprintf("file: %s", ProxyFile))
		readFile()
	} else if RedisKey != "" {
		logger.Println(fmt.Sprintf("redisKey: %s", RedisKey))
		rdb = redis.NewClient(&redis.Options{
			Addr:     RedisHost,
			Password: RedisPassword,
			DB:       RedisDb,
		})
	}
}

func main() {
	l, err := net.Listen("tcp", ProxyHost)
	if err != nil {
		logger.Panic(err)
	}
	go func() {
		if RedisKey != "" {
			if !IsServer {
				_, err := rdb.SAdd(RedisKey, ProxyServer).Result()
				if err != nil {
					logger.Println(err)
					return
				}
			}
		}
	}()
	updateProxies()
	changeProxy()
	for {
		client, err := l.Accept()
		if err != nil {
			logger.Println(err)
		}
		go handle(client)
		changeProxy()
	}
}

func readFile() {
	f, err := os.ReadFile(ProxyFile)
	if err != nil {
		logger.Println(err)
	}
	f = bytes.TrimSpace(f)
	Proxies = strings.Split(string(f), "\n")
	logger.Println("proxies count: ", len(Proxies))
}

func readRedis() {
	proxies, err := rdb.SMembers(RedisKey).Result()
	if err != nil {
		logger.Println(err)
		return
	}
	Proxies = proxies
	logger.Println("proxies count: ", len(Proxies))
}

func changeProxy() {
	go func() {
		if IsServer {
			if len(Proxies) == 0 {
				if ProxyFile != "" {
					readFile()
				} else if RedisKey != "" {
					readRedis()
				}
			}
			rand.Seed(time.Now().UnixNano())
			proxyServer := Proxies[rand.Intn(len(Proxies))]
			ProxyServer = proxyServer
		}
	}()
}

func updateProxies() {
	go func() {
		if IsServer {
			ticker := time.NewTicker(time.Hour)
			for {
				<-ticker.C
				if ProxyFile != "" {
					readFile()
				} else if RedisKey != "" {
					readRedis()
				}
			}
		}
	}()
}

func handle(client net.Conn) {
	if client == nil {
		logger.Println("client not ok")
		return
	}

	defer func() {
		err := client.Close()
		if err != nil {
			logger.Println(err)
			return
		}
	}()

	var b [2048]byte
	n, err := client.Read(b[:])
	if err != nil {
		logger.Println(err)
		return
	}
	var method, URL, address string
	if bytes.IndexByte(b[:], '\n') == -1 {
		logger.Println("headers empty")
		return
	}

	if Auth != "" {
		if !bytes.Contains(b[:], []byte(Auth)) {
			logger.Println("Authorization failed")
			return
		}
	}

	_, err = fmt.Sscanf(string(b[:bytes.IndexByte(b[:], '\n')]), "%s%s", &method, &URL)
	if err != nil {
		logger.Println(err)
		return
	}

	hostPortURL, err := url.Parse(URL)
	if err != nil {
		logger.Println(err)
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

	var proxyUser = ProxyUser
	var proxyPassword = ProxyPassword

	//拨号
	if IsServer {
		//logger.Println("ProxyAgent:", ProxyServer)
		address = ProxyServer
		arr := strings.Split(ProxyServer, "@")
		if len(arr) == 2 {
			address = arr[1]
			arr2 := strings.Split(arr[0], ":")
			proxyUser = arr2[0]
			proxyPassword = arr2[1]
		}
	}
	dialer := net.Dialer{Timeout: time.Minute}
	server, err := dialer.Dial("tcp", address)
	if err != nil {
		logger.Println(err)
		return
	}
	if method == "CONNECT" {
		if IsServer {
			setProxyHeader(b, server, proxyUser, proxyPassword)
		} else {
			_, err = fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
		}
		if err != nil {
			logger.Println(err)
			return
		}
	} else {
		if IsServer {
			setProxyHeader(b, server, proxyUser, proxyPassword)
		} else {
			_, err = server.Write(b[:n])
		}
		if err != nil {
			logger.Println(err)
			return
		}
	}

	//转发
	go func() {
		_, err = io.Copy(server, client)
		if err != nil {
			//use of closed network connection
			//logger.Println(err)
			return
		}
	}()
	_, err = io.Copy(client, server)
	if err != nil {
		logger.Println(err)
		return
	}
}

func setProxyHeader(b [2048]byte, server net.Conn, proxyUser string, proxyPassword string) {
	var proxyAuthorization = []byte(fmt.Sprintf("Proxy-Authorization: Basic %s\r\n\r\n", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxyUser, proxyPassword)))))
	var header []byte
	for _, v := range b {
		if v == 0 {
			break
		}
		header = append(header, v)
	}
	header = header[:len(header)-2]
	for _, v := range proxyAuthorization {
		header = append(header, v)
	}
	_, _ = server.Write(header)
}
