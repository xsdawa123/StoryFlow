package utils

import (
"gopkg.in/yaml.v3"
"os"
"go.uber.org/zap"
)

// Config 全局配置结构
type Config struct {
DB     DBConfig     `yaml:"DB"`
Server ServerConfig `yaml:"Server"`
}

// DBConfig 数据库配置
type DBConfig struct {
DSN string `yaml:"DSN"`
}

// ServerConfig 服务配置
type ServerConfig struct {
Port string `yaml:"Port"`
}

var globalConfig Config
var Logger *zap.Logger // 原生zap.Logger

// LoadConfig 加载yaml配置文件
func LoadConfig(path string) error {
file, err := os.Open(path)
if err != nil {
return err
}
defer file.Close()

decoder := yaml.NewDecoder(file)
if err := decoder.Decode(&globalConfig); err != nil {
return err
}
return nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
return &globalConfig
}

// InitLogger 初始化zap日志（适配原生zap.Logger）
func InitLogger() {
var err error
Logger, err = zap.NewProduction()
if err != nil {
panic("初始化日志失败: " + err.Error())
}
}

// ZapError 封装错误为zap.Field
func ZapError(err error) zap.Field {
return zap.Error(err)
}

// ZapInfof 适配格式化日志（原生zap转格式化）
func ZapInfof(format string, v ...interface{}) {
Logger.Sugar().Infof(format, v...)
}

// ZapFatalf 适配格式化致命日志
func ZapFatalf(format string, v ...interface{}) {
Logger.Sugar().Fatalf(format, v...)
}
