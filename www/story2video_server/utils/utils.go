package utils

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"

    _ "github.com/go-sql-driver/mysql"
)

// AI_SERVER_BASE_URL AI Server的基础地址
const AI_SERVER_BASE_URL = "http://127.0.0.1:7500"

// DB 全局数据库连接（阿里云RDS）
var DB *sql.DB

// InitDB 初始化阿里云RDS数据库连接（仅保留唯一声明）
func InitDB() {
    // 阿里云RDS连接信息（已保留你的实际配置）
    dsn := "User1111:User1111@tcp(rm-2zen0c30b4hlg3i7n.mysql.rds.aliyuncs.com:3306)/db_story2video?charset=utf8mb4&parseTime=True&loc=Local"
    var err error
    DB, err = sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal("【数据库】连接失败: ", err)
    }

    // 设置连接池参数（优化稳定性）
    DB.SetMaxOpenConns(20)
    DB.SetMaxIdleConns(10)
    DB.SetConnMaxLifetime(3600)

    // 验证连接
    if err := DB.Ping(); err != nil {
        log.Fatal("【数据库】Ping失败: ", err)
    }
    log.Println("【数据库】阿里云RDS连接成功")
}

// PostJSON 发送JSON POST请求到AI Server，带详细日志
func PostJSON(path string, data interface{}, resp interface{}) error {
    // 拼接完整URL
    fullURL := AI_SERVER_BASE_URL + path

    // 序列化请求体
    reqBody, err := json.Marshal(data)
    if err != nil {
        log.Printf("【AI Server】序列化失败 | URL: %s | 错误: %v | 请求数据: %+v", fullURL, err, data)
        return err
    }

    log.Printf("【AI Server】发送POST | URL: %s | 请求体: %s", fullURL, string(reqBody))

    // 发送请求
    req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(reqBody))
    if err != nil {
        log.Printf("【AI Server】创建请求失败 | URL: %s | 错误: %v", fullURL, err)
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    httpResp, err := client.Do(req)
    if err != nil {
        log.Printf("【AI Server】发送失败 | URL: %s | 错误: %v", fullURL, err)
        return err
    }
    defer httpResp.Body.Close()

    // 读取响应体
    respBody, err := io.ReadAll(httpResp.Body)
    if err != nil {
        log.Printf("【AI Server】读取响应失败 | URL: %s | 错误: %v", fullURL, err)
        return err
    }

    log.Printf("【AI Server】响应 | 状态码: %d | URL: %s | 响应体: %s", httpResp.StatusCode, fullURL, string(respBody))

    // 反序列化响应体（容错）
    if err := json.Unmarshal(respBody, resp); err != nil {
        log.Printf("【AI Server】反序列化失败 | URL: %s | 错误: %v | 响应体: %s", fullURL, err, string(respBody))
        return fmt.Errorf("AI Server返回非JSON响应: %s", string(respBody))
    }

    return nil
}
