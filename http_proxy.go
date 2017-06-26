package main

import (
    "log"
    "net"
    "net/http"
    "net/http/httputil"
    "net/url"
    "os"
    "encoding/json"
    "fmt"
    "time"
)

type Client struct {
    time int64
    count int
}

type handle struct {
    Lhost string          `json:"local_host"`
    Rhost string            `json:"remote_host"`
    Length int          `json:"max_length"`
    Urlmap map[string]string `json:"url_map"`
    black_list map[string]bool
    count_map map[string]*Client
}

var h handle = handle{
    Lhost: "",
    Rhost: "",
    Length: 1024,
    Urlmap: map[string]string{},
    black_list: map[string]bool{},
    count_map: map[string]*Client{},
}



func readCfg(path string) {
    file, err := os.Open(path)
    if err != nil {
        panic("fail to read config file: " + path)
        return
    }


    fi, _ := file.Stat()

    buff := make([]byte, fi.Size())
    _, err = file.Read(buff)
    buff = []byte(os.ExpandEnv(string(buff)))

    err = json.Unmarshal(buff, &h)
    if err != nil {
        log.Print(err)
        panic("failed to unmarshal file")
        return
    }
    log.Print(h)
}

func (this *handle) CheckIP(ip string) bool {
    _,err := h.black_list[ip]
    if err {
        return false
    }

    now := time.Now().Unix()
    c,err1 := h.count_map[ip]
    if !err1 {
        h.count_map[ip] = &Client{time: now, count: 1}
    } else {
        if(now/60 == c.time/60) {
            c.count++
            if c.count > 20 {
                h.black_list[ip] = true
                log.Printf("ip:%s add to black list", ip)
                return false
            }
        } else {
            c.time = now
            c.count = 1
        }
    }
    return true
}

func (this *handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Println("handle request")
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        fmt.Println("ip切割非法")
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }

    if !h.CheckIP(ip) {
        fmt.Println("ip检查非法")
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }

    _, err1 := h.Urlmap[r.URL.Path]
    if !err1 {
        fmt.Println("url检查非法")
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }

    remote, err2 := url.Parse(h.Rhost)
    if err2 != nil {
        fmt.Println("客户端地址获取非法")
        h.black_list[ip] = true
        http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
        return
    }
    fmt.Println("代理开始")
    proxy := httputil.NewSingleHostReverseProxy(remote)
    proxy.ServeHTTP(w, r)
}

func startServer() {
    err := http.ListenAndServe(h.Lhost, &h)
    if err != nil {
        fmt.Printf("ListenAndServe: ", err)
    }
}

func main() {
    if len(os.Args) != 2 {
        panic("please identify a config file")
        return
    }

    readCfg(os.Args[1])

    startServer()
}
