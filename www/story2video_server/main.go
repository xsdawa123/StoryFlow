package main

import (
"story2video_server/db"
"story2video_server/handler"
"story2video_server/utils"
"github.com/gin-contrib/cors"
"github.com/gin-gonic/gin"
)

func main() {
// 加载配置文件
if err := utils.LoadConfig("config/config.yaml"); err != nil {
panic("加载配置失败: " + err.Error())
}

// 初始化日志
utils.InitLogger()

// 初始化数据库
db.InitDB()

// 初始化Gin引擎
r := gin.Default()

// 跨域配置（允许前端调用）
r.Use(cors.New(cors.Config{
AllowOrigins:     []string{"*"},
AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
AllowHeaders:     []string{"Content-Type"},
AllowCredentials: true,
}))

// 客户端接口路由
clientGroup := r.Group("/api/client")
{
clientGroup.POST("/story/generate", handler.GenerateStory)
clientGroup.POST("/images/generate", handler.GenerateImages)
clientGroup.POST("/videos/generate", handler.GenerateVideos)
clientGroup.GET("/shots/list", handler.ListShots)
clientGroup.GET("/project/list", handler.ListProjects)
}

// AI回调接口路由
callbackGroup := r.Group("/api/callback")
{
callbackGroup.POST("/story/result", handler.StoryResultCallback)
callbackGroup.POST("/image/result", handler.ImageResultCallback)
callbackGroup.POST("/video/result", handler.VideoResultCallback)
}

// 启动服务
port := utils.GetConfig().Server.Port
utils.ZapInfof("服务启动，监听端口: %s", port)
if err := r.Run(port); err != nil {
utils.ZapFatalf("服务启动失败: %v", err)
}
}
