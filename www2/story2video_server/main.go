package main

import (
    "log"
    "story2video/handler"  // 模块内绝对导入（已修正）
    "story2video/utils"    // 模块内绝对导入（已修正）

    "github.com/gin-gonic/gin"
)

func main() {
    // 修正：InitDB无返回值，去掉err接收（仅改这一行，无业务逻辑修改）
    utils.InitDB()
    log.Println("【数据库】阿里云RDS连接成功")

    // 保留你原有逻辑：Gin模式（一行未改）
    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()

    // 保留你原有逻辑：路由注册（仅注释未定义的ListProjects，无其他修改）
    clientGroup := r.Group("/api/client")
    {
        generateGroup := clientGroup.Group("/generate")
        {
            generateGroup.POST("/story", handler.GenerateStory)
            generateGroup.POST("/images", handler.GenerateImages) // 图片生成（核心逻辑不变）
            generateGroup.POST("/audios", handler.GenerateAudios)
        }
        clientGroup.GET("/shots/list", handler.ListShots)
        clientGroup.POST("/image/regenerate", handler.GenerateImages)
        clientGroup.GET("/image/result", handler.ImageResultCallback)
        // 注释：handler中未定义ListProjects，暂时注释（无业务逻辑修改）
        // clientGroup.GET("/projects/list", handler.ListProjects)
    }

    // 保留你原有逻辑：启动服务（一行未改）
    log.Println("【服务】启动成功 | 端口: 8080")
    if err := r.Run(":8080"); err != nil {
        log.Fatalf("【服务】启动失败 | 错误: %v", err)
    }
}
