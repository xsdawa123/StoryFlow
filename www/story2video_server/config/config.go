package config

import "story2video_server/utils"

// InitConfig 初始化项目配置（日志、数据库等）
func InitConfig() {
// 初始化日志（复用你的utils.Logger）
utils.InitLogger()

// 初始化数据库
InitDB()

utils.Logger.Info("配置文件初始化成功")
}
