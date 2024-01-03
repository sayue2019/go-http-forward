package main

import (
    "bytes"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "os"
)

// Config 结构体用于存储配置信息
type Config struct {
    ForwardURL     string
    ListenPort     string
    AccessPassword string
}

var config Config

func main() {
    // 从环境变量中读取配置，如果环境变量为空，则使用默认值
    config.ForwardURL = getEnv("FORWARD_URL", "http://127.0.0.1:19088")
    config.ListenPort = getEnv("LISTEN_PORT", "5999")
    config.AccessPassword = getEnv("ACCESS_PASSWORD", "wechat5999")

    // 检查端口是否已被占用
    ln, err := net.Listen("tcp", ":"+config.ListenPort)
    if err != nil {
        log.Fatalf("无法监听端口 %s: %v", config.ListenPort, err)
    }
    defer ln.Close()

    // 设置HTTP服务器监听的端口
    http.HandleFunc("/", handler)
    log.Println("服务器启动，监听端口：" + config.ListenPort)
    log.Fatal(http.Serve(ln, nil)) // 使用ln作为监听器
}

// getEnv 从环境变量中获取值，如果未设置则返回默认值
func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}

func handler(w http.ResponseWriter, r *http.Request) {
    // 验证访问密码
    password := r.URL.Query().Get("password")
    if password != config.AccessPassword {
        http.Error(w, "无效的访问密码", http.StatusUnauthorized)
        return
    }

    // 根据请求类型处理
    switch r.Method {
    case "GET", "POST":
        forwardRequest(w, r)
    default:
        http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
    }
}

func forwardRequest(w http.ResponseWriter, r *http.Request) {
    // 创建新的请求
    forwardURL := config.ForwardURL + r.URL.RequestURI()
    req, err := http.NewRequest(r.Method, forwardURL, nil)
    if err != nil {
        http.Error(w, "创建请求错误: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // 复制原始请求的Header和Body（如果是POST方法）
    req.Header = r.Header
    if r.Method == "POST" {
        body, err := ioutil.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "读取请求体错误: "+err.Error(), http.StatusInternalServerError)
            return
        }
        defer r.Body.Close()
        req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
    }

    // 发送请求
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        http.Error(w, "转发请求错误: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    // 将目标服务的响应转发回原始客户端
    response, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, "读取响应错误: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(response)
}
