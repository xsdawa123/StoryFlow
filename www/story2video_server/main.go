package main

import (
    "github.com/gin-gonic/gin"
    "story2video_server/handler" // 替换为你实际的handler包路径（如果不同请修改）
    "story2video_server/utils"
)

func main() {
    utils.InitDB()
    // 初始化Gin引擎（生产环境可改为gin.ReleaseMode）
    gin.SetMode(gin.DebugMode)
    r := gin.Default()

    // 注册/api/client前缀的路由组（核心：匹配前端请求层级）
    clientGroup := r.Group("/api/client")
    {
        // 注册/generate子组（匹配 /api/client/generate/xxx 路径）
        generateGroup := clientGroup.Group("/generate")
        {
            // 故事生成：POST /api/client/generate/story（匹配前端请求）
            generateGroup.POST("/story", handler.GenerateStory)
            // 图片生成：POST /api/client/generate/images
            generateGroup.POST("/images", handler.GenerateImages)
            // 音频生成：POST /api/client/generate/audios
            generateGroup.POST("/audios", handler.GenerateAudios)
        }

        // 分镜列表：GET /api/client/shots/list
        clientGroup.GET("/shots/list", handler.ListShots)
        // 图片重生成（占位）：POST /api/client/image/regenerate
        clientGroup.POST("/image/regenerate", handler.GenerateImages)
        // 图片结果查询（占位）：GET /api/client/image/result
        clientGroup.GET("/image/result", handler.ImageResultCallback)
    }

    // 关键：绑定到0.0.0.0:8080（允许外部IP访问，而非仅本地127.0.0.1）
    err := r.Run("0.0.0.0:8080")
    if err != nil {
        panic("服务启动失败：" + err.Error())
    }
}
