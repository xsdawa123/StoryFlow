package main

import (
"log"
"story2video_server/handler"
"story2video_server/utils"

"github.com/gin-gonic/gin"
)

func main() {
// 初始化数据库
utils.InitDB()

// 创建Gin引擎
r := gin.Default()

// 注册路由
api := r.Group("/api")
{
// 原有接口（不改动）
api.POST("/generate/story", handler.GenerateStory)
api.POST("/generate/images", handler.GenerateImages)
api.POST("/generate/audios", handler.GenerateAudios)
api.GET("/shots/list", handler.ListShots)
api.GET("/projects/list", handler.ListProjects)

// Client接口组（对应你的需求）
client := api.Group("/client")
{
client.POST("/story/generate", handler.StoryGenerate)  // 1. 创建故事
client.GET("/shots/list", handler.ClientListShots)     // 2. 轮询分镜列表
client.POST("/image/regenerate", handler.ImageRegenerate) // 3. 重新生成图片
client.GET("/image/result", handler.ImageResult)       // 4. 轮询图片状态
}
}

// 启动服务
log.Println("服务启动成功，监听端口8080")
if err := r.Run(":8080"); err != nil {
log.Fatal("服务启动失败: ", err)
}
}
