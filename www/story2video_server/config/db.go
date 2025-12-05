package config

import (
"gorm.io/driver/mysql"
"gorm.io/gorm"
"story2video_server/utils"
"go.uber.org/zap" // 必须导入zap包，用于zap.Error/zap.String
)

// DB 全局数据库连接对象
var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() {
// 数据库DSN（匹配你的RDS信息）
dsn := "User1111:User1111@tcp(rm-2zen0c30b4hlg3i7n.mysql.rds.aliyuncs.com:3306)/db_story2video?charset=utf8mb4&parseTime=True&loc=Local"

var err error
DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
if err != nil {
// 正确写法：zap.Error(err) 生成zap.Field类型
utils.Logger.Fatal("数据库连接失败", zap.Error(err))
panic("数据库连接失败：" + err.Error())
}

// 正确写法：zap.String("key", "value") 生成zap.Field类型
utils.Logger.Info("数据库连接成功",
zap.String("host", "rm-2zen0c30b4hlg3i7n.mysql.rds.aliyuncs.com"),
zap.String("dbname", "db_story2video"),
)
}
