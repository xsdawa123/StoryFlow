package db

import (
"database/sql"
_ "github.com/go-sql-driver/mysql"
"story2video_server/utils"
)

// DB 全局数据库连接
var DB *sql.DB

// InitDB 初始化数据库连接
func InitDB() {
// 从配置中读取DSN
dsn := utils.GetConfig().DB.DSN
var err error
DB, err = sql.Open("mysql", dsn)
if err != nil {
utils.Logger.Fatal("数据库连接失败", utils.ZapError(err))
}

// 测试连接
if err := DB.Ping(); err != nil {
utils.Logger.Fatal("数据库ping失败", utils.ZapError(err))
}
utils.Logger.Info("数据库连接成功")
}
