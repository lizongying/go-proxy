package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	"io"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var IsServer bool

var ProxyServer *url.URL
var Proxies []*url.URL

var ProxyFile string
var RedisKey string

var logger *log.Logger
var rdb *redis.Client

var Auth string

const maxTimeout = 60   // second
const maxRequest = 2048 // byte

var reProxyAuthorization = regexp.MustCompile(`Proxy-Authorization: (Basic [^\r]+)\r\n`)
var reProxyConnection = regexp.MustCompile(`Proxy-Connection: `)

func init() {
	proxyHost := flag.String("ph", "127.0.0.1:8081", "proxyHost")
	proxyFile := flag.String("pf", "", "proxyFile")
	redisHost := flag.String("rh", "127.0.0.1:6379", "redisHost")
	redisPassword := flag.String("rp", "", "redisPassword")
	redisDb := flag.Int("rd", 0, "redisDb")
	redisKey := flag.String("rk", "", "redisKey")
	isServer := flag.Bool("s", false, "isServer")
	isDebug := flag.Bool("v", false, "isDebug")
	isQuiet := flag.Bool("q", false, "isQuiet")
	flag.Parse()
	ProxyHost := *proxyHost
	ProxyFile = *proxyFile
	RedisHost := *redisHost
	RedisPassword := *redisPassword
	RedisDb := *redisDb
	RedisKey = *redisKey
	IsServer = *isServer
	IsDebug := *isDebug
	IsQuiet := *isQuiet
	if IsDebug {
		logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	}
	if IsQuiet {
		logger.SetOutput(io.Discard)
	}

	proxyServer, err := url.Parse(ProxyHost)
	if err != nil {
		logger.Panic(err)
	}
	ProxyServer = proxyServer
	user := ProxyServer.User.String()
	if ProxyServer.Scheme == "" {
		ProxyServer.Scheme = "http"
	}
	if ProxyServer.Hostname() == "" {
		ProxyServer.Host = fmt.Sprintf("%s:%s", "127.0.0.1", ProxyServer.Port())
	}
	if ProxyServer.Port() == "" {
		ProxyServer.Host = fmt.Sprintf("%s:%s", ProxyServer.Hostname(), "8081")
	}
	if ProxyServer.User.String() != "" {
		Auth = base64.StdEncoding.EncodeToString([]byte(user))
	}

	if IsServer {
		logger.Println("mode: server")
	} else {
		logger.Println("mode: client")
	}

	logger.Println("proxy:", ProxyServer.String())

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
	var err error

	Port := ProxyServer.Port()
	var listener net.Listener
	if ProxyServer.Scheme == "https" {
		l, e := net.Listen("tcp", fmt.Sprintf(":%s", Port))
		listener = l
		err = e
	} else {
		l, e := net.Listen("tcp", fmt.Sprintf(":%s", Port))
		listener = l
		err = e
	}
	if err != nil {
		logger.Panic(err)
	}
	go func() {
		if rdb != nil {
			if !IsServer {
				_, err = rdb.SAdd(RedisKey, ProxyServer.String()).Result()
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
		conn, e := listener.Accept()
		if e != nil {
			err = e
			logger.Println(err)
			continue
		}
		go handle(conn)
		changeProxy()
	}
}

func readFile() {
	f, err := os.ReadFile(ProxyFile)
	if err != nil {
		logger.Println(err)
	}
	f = bytes.TrimSpace(f)
	var ProxiesNew []*url.URL
	proxies := strings.Split(string(f), "\n")
	for _, v := range proxies {
		u, _ := url.Parse(v)
		if u.Host != "" {
			ProxiesNew = append(ProxiesNew, u)
		}
	}
	Proxies = ProxiesNew
	logger.Println("proxies count: ", len(Proxies))
}

func readRedis() {
	proxies, err := rdb.SMembers(RedisKey).Result()
	if err != nil {
		logger.Println(err)
		return
	}
	var ProxiesNew []*url.URL
	for _, v := range proxies {
		u, _ := url.Parse(v)
		if u.Host != "" {
			ProxiesNew = append(ProxiesNew, u)
		}
	}
	Proxies = ProxiesNew
	logger.Println("proxies count: ", len(Proxies))
}

func changeProxy() {
	go func() {
		if IsServer {
			if len(Proxies) == 0 {
				if ProxyFile != "" {
					readFile()
				} else if rdb != nil {
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
				} else if rdb != nil {
					readRedis()
				}
			}
		}
	}()
}

func handle(conn net.Conn) {
	var err error

	if conn == nil {
		err = errors.New("conn not ok")
		logger.Println(err)
		return
	}

	defer func() {
		err = conn.Close()
		if err != nil {
			logger.Println(err)
			return
		}
	}()

	var b [maxRequest]byte
	n, err := conn.Read(b[:])
	if err != nil {
		logger.Println(err)
		return
	}

	body := b[:n]
	var method, URL, version string
	headerIndex := bytes.IndexByte(body, '\n') + 1
	if headerIndex == 0 {
		err = errors.New("headers empty")
		logger.Println(err)
		return
	}

	if Auth != "" {

		// note: maybe Authorization not Proxy-Authorization
		if !bytes.Contains(body, []byte(Auth)) {
			err = errors.New("authorization failed")
			logger.Println(err)
			return
		}
	}

	_, err = fmt.Sscanf(string(body[:headerIndex]), "%s%s%s", &method, &URL, &version)
	if err != nil {
		logger.Println(err)
		return
	}

	if !strings.Contains(URL, "//") {
		URL = fmt.Sprintf("//%s", URL)
	}
	address, err := url.Parse(URL)
	if err != nil {
		logger.Println(err)
		return
	}
	if address.Host == "false" {
		logger.Println("host false")
		return
	}

	if method == "CONNECT" {
		address.Scheme = "https"
	} else {
		if address.Port() == "" {
			address.Host = fmt.Sprintf("%s:80", address.Host)
		}
	}

	//拨号
	if IsServer {
		address = ProxyServer
	}
	dialer := net.Dialer{Timeout: time.Second * time.Duration(maxTimeout)}

	var connDest net.Conn
	defer func() {
		if connDest != nil {
			err = connDest.Close()
			if err != nil {
				logger.Println(err)
				return
			}
		}
	}()

	if address.Scheme == "https" && IsServer {
		s, e := dialer.Dial("tcp", address.Host)
		if e != nil {
			err = e
			logger.Println(err)
			return
		}
		connDest = s
	} else {
		s, e := dialer.Dial("tcp", address.Host)
		if e != nil {
			err = e
			logger.Println(err)
			return
		}
		connDest = s
	}

	// TODO 应该复用 http2?
	user := address.User.String()
	if user != "" {

		// just for server
		setProxyHeader(&body, headerIndex, connDest, user)
	}

	if method == "CONNECT" {
		_, err = fmt.Fprint(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
		if err != nil {
			logger.Println(err)
			return
		}
	} else {

		// request changed, should do something
		if !IsServer {
			setHeader(&body, headerIndex, method, version)
		}
		_, err = connDest.Write(body)
		if err != nil {
			logger.Println(err)
			return
		}
	}

	go func() {
		_, err = io.Copy(connDest, conn)
		if err != nil {
			//use of closed network connection
			//logger.Println(err)
			return
		}
	}()
	_, err = io.Copy(conn, connDest)
	if err != nil {
		logger.Println(err)
		return
	}
}

func setProxyHeader(body *[]byte, headerIndex int, conn net.Conn, user string) {
	var proxyAuthorization = []byte(fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", base64.StdEncoding.EncodeToString([]byte(user))))
	*body = reProxyAuthorization.ReplaceAll(*body, []byte(nil))
	bodyNew := (*body)[:headerIndex]
	bodyAfter := []byte(string((*body)[headerIndex:]))

	// add new proxy
	bodyNew = append(bodyNew, proxyAuthorization...)
	bodyNew = append(bodyNew, bodyAfter...)
	_, err := conn.Write(bodyNew)
	if err != nil {
		logger.Println(err)
	}
	*body = bodyNew
}

func setHeader(body *[]byte, headerIndex int, method string, version string) {
	bodyNew := []byte(fmt.Sprintf("%s / %s\r\n", method, version))
	bodyNew = append(bodyNew, (*body)[headerIndex:]...)
	bodyNew = reProxyAuthorization.ReplaceAll(bodyNew, []byte(nil))
	bodyNew = reProxyConnection.ReplaceAll(bodyNew, []byte(`Connection: `))
	*body = bodyNew
}
