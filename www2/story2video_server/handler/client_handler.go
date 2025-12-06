package handler

import (
"io"
"log"
"net/http"
"os"

"story2video/utils"

"github.com/gin-gonic/gin"
)

// 初始化日志文件
func init() {
logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
log.Fatal("创建日志文件失败: ", err)
}
log.SetOutput(io.MultiWriter(os.Stdout, logFile))
log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// -------------------------- 分镜生成（严格对齐story_request格式）--------------------------
func GenerateStory(c *gin.Context) {
// 文档要求：请求参数 = project_id + raw_story + style（对应projects表字段）
type StoryGenerateReq struct {
ProjectID string `json:"project_id" binding:"required"` // projects.id
RawStory  string `json:"raw_story" binding:"required"`  // projects.raw_story
Style     string `json:"style" binding:"required"`      // projects.style
UserID    string `json:"user_id" binding:"required"`
}

var req StoryGenerateReq
if err := c.ShouldBindJSON(&req); err != nil {
log.Printf("【分镜生成】解析参数失败: %v | 请求参数: %+v", err, req)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

log.Printf("【分镜生成】解析参数成功 | 项目ID: %s | 故事内容: %s | 风格: %s | 用户ID: %s",
req.ProjectID, req.RawStory, req.Style, req.UserID)

// 构造AI Server请求（完全对齐文档story_request格式）
aiReq := map[string]interface{}{
"project_id": req.ProjectID,
"raw_story":  req.RawStory,
"style":      req.Style,
"callback_url": "http://127.0.0.1:8080/api/callback/story/result",
}

log.Printf("【分镜生成】准备调用AI Server | 路径: /api/generate/story | 请求参数: %+v", aiReq)

// 调用AI Server（响应格式：project_id + shot_1/shot_2...）
var aiResp map[string]interface{}
err := utils.PostJSON("/api/generate/story", aiReq, &aiResp)
if err != nil {
log.Printf("【分镜生成】调用AI Server失败 | 路径: /api/generate/story | 错误: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "调用AI Server失败: " + err.Error(),
})
return
}

log.Printf("【分镜生成】调用AI Server成功 | 返回数据: %+v", aiResp)

// 写入projects表（初始化项目状态）
if utils.DB != nil {
_, err = utils.DB.Exec(`
INSERT INTO projects (project_id, user_id, raw_story, style, project_status, create_time, update_time)
VALUES (?, ?, ?, ?, 'generated', NOW(), NOW())
ON DUPLICATE KEY UPDATE project_status='generated', update_time=NOW()
`, req.ProjectID, req.UserID, req.RawStory, req.Style)
if err != nil {
log.Printf("【分镜生成】写入projects表失败 | 项目ID: %s | 错误: %v", req.ProjectID, err)
} else {
log.Printf("【分镜生成】写入projects表成功 | 项目ID: %s", req.ProjectID)
}

// 解析shot_*字段，写入shots表（对齐文档story_response与shots表映射）
projectID := aiResp["project_id"].(string)
for key, value := range aiResp {
if len(key) >= 5 && key[:5] == "shot_" {
shotMap, ok := value.(map[string]interface{})
if !ok {
log.Printf("【分镜生成】shot格式错误 | key: %s | 值: %+v", key, value)
continue
}

// 文档字段映射：shot_response -> shots表
sceneTitle := shotMap["scene_title"].(string)
prompt := shotMap["prompt"].(string)
narration := shotMap["narration"].(string)
transition := shotMap["transition"].(string)
shotStatus := shotMap["shot_status"].(string)

// shots.id自动生成（文档要求），无需传入
_, err = utils.DB.Exec(`
INSERT INTO shots (project_id, scene_title, prompt, narration, transition, shot_status, create_time, update_time)
VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE update_time=NOW()
`, projectID, sceneTitle, prompt, narration, transition, shotStatus)
if err != nil {
log.Printf("【分镜生成】写入shots表失败 | 项目ID: %s | shot: %s | 错误: %v", projectID, key, err)
} else {
log.Printf("【分镜生成】写入shots表成功 | 项目ID: %s | shot: %s", projectID, key)
}
}
}
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": aiResp,
"msg":  "分镜生成任务已完成",
})
}

// -------------------------- 图片生成（严格对齐image_request格式）--------------------------
func GenerateImages(c *gin.Context) {
// 文档要求：请求参数 = project_id + shot_1/shot_2...（每个shot含shot_id/prompt/style）
type Shot struct {
ShotID string `json:"shot_id" binding:"required"` // shots.id
Prompt string `json:"prompt" binding:"required"`  // shots.prompt
Style  string `json:"style" binding:"required"`   // projects.style
}
type ImageGenerateReq struct {
ProjectID string          `json:"project_id" binding:"required"` // projects.id
UserID    string          `json:"user_id" binding:"required"`
Shots     map[string]Shot `json:"-"` // 手动解析shot_1/shot_2动态字段
}

// 解析原始JSON，适配shot_*动态字段
var rawReq map[string]interface{}
if err := c.ShouldBindJSON(&rawReq); err != nil {
log.Printf("【图片生成】解析参数失败: %v | 原始请求: %+v", err, rawReq)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

// 校验必填参数
projectID, ok := rawReq["project_id"].(string)
if !ok || projectID == "" {
log.Printf("【图片生成】project_id缺失 | 原始请求: %+v", rawReq)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "project_id是必填参数",
})
return
}
userID, ok := rawReq["user_id"].(string)
if !ok || userID == "" {
log.Printf("【图片生成】user_id缺失 | 原始请求: %+v", rawReq)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "user_id是必填参数",
})
return
}

// 提取shot_*字段，构造AI Server请求（对齐image_request格式）
aiReq := make(map[string]interface{})
aiReq["project_id"] = projectID
hasValidShot := false
for key, value := range rawReq {
if len(key) >= 5 && key[:5] == "shot_" {
shotMap, ok := value.(map[string]interface{})
if !ok {
log.Printf("【图片生成】shot格式错误 | key: %s | 值: %+v", key, value)
continue
}
// 校验shot必填字段
shotID, _ := shotMap["shot_id"].(string)
prompt, _ := shotMap["prompt"].(string)
style, _ := shotMap["style"].(string)
if shotID == "" || prompt == "" || style == "" {
log.Printf("【图片生成】shot字段缺失 | key: %s | 字段: %+v", key, shotMap)
continue
}
aiReq[key] = shotMap
hasValidShot = true
}
}
if !hasValidShot {
log.Printf("【图片生成】无有效shot字段 | 原始请求: %+v", rawReq)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "至少需要一个有效shot_*字段（含shot_id/prompt/style）",
})
return
}
aiReq["callback_url"] = "http://127.0.0.1:8080/api/callback/image/result"

log.Printf("【图片生成】准备调用AI Server | 路径: /api/generate/images | 请求参数: %+v", aiReq)

// 调用AI Server（响应格式：project_id + shot_1/shot_2...（含image_url））
var aiResp map[string]interface{}
err := utils.PostJSON("/api/generate/images", aiReq, &aiResp)
if err != nil {
log.Printf("【图片生成】调用AI Server失败 | 路径: /api/generate/images | 错误: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "调用AI Server失败: " + err.Error(),
})
return
}

log.Printf("【图片生成】调用AI Server成功 | 返回数据: %+v", aiResp)

// 写入shots表（更新image_url和shot_status，对齐文档image_response映射）
// 写入shots表（更新image_url和shot_status，对齐文档image_response映射）
if utils.DB != nil {
    for key, value := range aiResp {
        if len(key) >= 5 && key[:5] == "shot_" {
            // 从shot_1提取shotID（解决AI返回shot_id为空）
            shotIDFromKey := key[5:]
            shotMap, ok := value.(map[string]interface{})
            if !ok {
                continue
            }
            // 安全提取字段，防止nil panic
            shotID := shotIDFromKey // 优先用key提取的ID
            if sID, ok := shotMap["shot_id"].(string); ok && sID != "" {
                shotID = sID
            }
            imageURL := ""
            if imgURL, ok := shotMap["image_url"].(string); ok {
                imageURL = imgURL
            }
            shotStatus := "generated"
            if sStatus, ok := shotMap["shot_status"].(string); ok {
                shotStatus = sStatus
            }
            // 修正UPDATE条件：id → shot_id
            _, err = utils.DB.Exec(`
                UPDATE shots SET image_url=?, shot_status=?, update_time=NOW()
                WHERE project_id=? AND shot_id=?
            `, imageURL, shotStatus, projectID, shotID)
            if err != nil {
                // 更新失败则插入，兜底保证URL写入
                log.Printf("【图片生成】更新失败，尝试插入 | projectID: %s | shotID: %s", projectID, shotID)
                _, insErr := utils.DB.Exec(`
                    INSERT INTO shots (shot_id, project_id, image_url, shot_status)
                    VALUES (?, ?, ?, ?)
                    ON DUPLICATE KEY UPDATE image_url=?, shot_status=?
                `, shotID, projectID, imageURL, shotStatus, imageURL, shotStatus)
                if insErr != nil {
                    log.Printf("【图片生成】插入失败 | err: %v", insErr)
                }
            }
        }
    }
}
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": aiResp,
"msg":  "图片生成任务已完成",
})
}

// -------------------------- 音频生成（严格对齐audio_request格式）--------------------------
func GenerateAudios(c *gin.Context) {
// 文档要求：请求参数 = project_id + shot_1/shot_2...（每个shot含shot_id/narration）
type Shot struct {
ShotID    string `json:"shot_id" binding:"required"` // shots.id
Narration string `json:"narration" binding:"required"`// shots.narration
}
type AudioGenerateReq struct {
ProjectID string          `json:"project_id" binding:"required"` // projects.id
UserID    string          `json:"user_id" binding:"required"`
Shots     map[string]Shot `json:"-"` // 手动解析shot_*动态字段
}

// 解析原始JSON，适配shot_*动态字段
var rawReq map[string]interface{}
if err := c.ShouldBindJSON(&rawReq); err != nil {
log.Printf("【音频生成】解析参数失败: %v | 原始请求: %+v", err, rawReq)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

// 校验必填参数
projectID, ok := rawReq["project_id"].(string)
if !ok || projectID == "" {
log.Printf("【音频生成】project_id缺失 | 原始请求: %+v", rawReq)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "project_id是必填参数",
})
return
}
userID, ok := rawReq["user_id"].(string)
if !ok || userID == "" {
log.Printf("【音频生成】user_id缺失 | 原始请求: %+v", rawReq)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "user_id是必填参数",
})
return
}

// 提取shot_*字段，构造AI Server请求（对齐audio_request格式）
aiReq := make(map[string]interface{})
aiReq["project_id"] = projectID
hasValidShot := false
for key, value := range rawReq {
if len(key) >= 5 && key[:5] == "shot_" {
shotMap, ok := value.(map[string]interface{})
if !ok {
log.Printf("【音频生成】shot格式错误 | key: %s | 值: %+v", key, value)
continue
}
// 校验shot必填字段（文档要求：shot_id + narration）
shotID, _ := shotMap["shot_id"].(string)
narration, _ := shotMap["narration"].(string)
if shotID == "" || narration == "" {
log.Printf("【音频生成】shot字段缺失 | key: %s | 字段: %+v", key, shotMap)
continue
}
aiReq[key] = shotMap
hasValidShot = true
}
}
if !hasValidShot {
log.Printf("【音频生成】无有效shot字段 | 原始请求: %+v", rawReq)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "至少需要一个有效shot_*字段（含shot_id/narration）",
})
return
}
aiReq["callback_url"] = "http://127.0.0.1:8080/api/callback/audio/result"

log.Printf("【音频生成】准备调用AI Server | 路径: /api/generate/audios | 请求参数: %+v", aiReq)

// 调用AI Server（响应格式：project_id + shot_1/shot_2...（含audio_url））
var aiResp map[string]interface{}
err := utils.PostJSON("/api/generate/audios", aiReq, &aiResp)
if err != nil {
log.Printf("【音频生成】调用AI Server失败 | 路径: /api/generate/audios | 错误: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "调用AI Server失败: " + err.Error(),
})
return
}

log.Printf("【音频生成】调用AI Server成功 | 返回数据: %+v", aiResp)

// 写入shots表（更新audio_url和shot_status，对齐音频响应映射）
if utils.DB != nil {
    for key, value := range aiResp {
        if len(key) >= 5 && key[:5] == "shot_" {
            // 从shot_1/shot_2提取shotID（解决AI返回shot_id为空）
            shotIDFromKey := key[5:]
            shotMap, ok := value.(map[string]interface{})
            if !ok {
                continue
            }

            // 安全提取字段，防止nil panic
            shotID := shotIDFromKey // 优先用key提取的ID
            if sID, ok := shotMap["shot_id"].(string); ok && sID != "" {
                shotID = sID
            }
            audioURL := ""
            if audURL, ok := shotMap["audio_url"].(string); ok {
                audioURL = audURL
            }
            shotStatus := "generated"
            if sStatus, ok := shotMap["shot_status"].(string); ok {
                shotStatus = sStatus
            }

            // 修正UPDATE条件：id → shot_id，字段改为audio_url
            _, err = utils.DB.Exec(`
                UPDATE shots SET audio_url=?, shot_status=?, update_time=NOW()
                WHERE project_id=? AND shot_id=?
            `, audioURL, shotStatus, projectID, shotID)
            
            if err != nil {
                // 更新失败则插入，兜底保证URL写入
                log.Printf("【音频生成】更新失败，尝试插入 | projectID: %s | shotID: %s", projectID, shotID)
                _, insErr := utils.DB.Exec(`
                    INSERT INTO shots (shot_id, project_id, audio_url, shot_status)
                    VALUES (?, ?, ?, ?)
                    ON DUPLICATE KEY UPDATE audio_url=VALUES(audio_url), shot_status=VALUES(shot_status)
                `, shotID, projectID, audioURL, shotStatus)
                
                if insErr != nil {
                    log.Printf("【音频生成】插入失败 | err: %v", insErr)
                } else {
                    log.Printf("【音频生成】URL写入成功 | shotID: %s | URL: %s", shotID, audioURL)
                }
            } else {
                log.Printf("【音频生成】URL写入成功 | shotID: %s | URL: %s", shotID, audioURL)
            }
        }
    }
}
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": aiResp,
"msg":  "音频生成任务已完成",
})
}

// -------------------------- 列表查询接口（新增整体status判断，语法完整）--------------------------
func ListShots(c *gin.Context) {
    projectID := c.Query("project_id")
    if projectID == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "code": 400,
            "data": gin.H{
                "status": "fail",
                "project_data": gin.H{"storyboards": []interface{}{}},
                "overall_status": "running",
            },
            "msg":  "project_id是必填参数",
        })
        return
    }
    if utils.DB == nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "code": 500,
            "data": gin.H{
                "status": "error",
                "project_data": gin.H{"storyboards": []interface{}{}},
                "overall_status": "running",
            },
            "msg":  "数据库连接未初始化",
        })
        return
    }

    // 数据库查询逻辑
    rows, err := utils.DB.Query(`
    SELECT shot_id, project_id, scene_title, prompt, narration, transition, image_url, audio_url, shot_status
    FROM shots WHERE project_id=?
    `, projectID)
    if err != nil {
        log.Printf("【查询分镜】数据库查询失败 | 项目ID: %s | 错误: %v", projectID, err)
        c.JSON(http.StatusInternalServerError, gin.H{
            "code": 500,
            "data": gin.H{
                "status": "error",
                "project_data": gin.H{"storyboards": []interface{}{}},
                "overall_status": "running",
            },
            "msg":  "查询分镜失败: " + err.Error(),
        })
        return
    }
    defer rows.Close()

    // 1. 遍历分镜并检查核心字段完整性
    var storyboards []map[string]interface{}
    isAllCompleted := true // 初始化：默认所有分镜都完整
    for rows.Next() {
        var shotID, pID, sceneTitle, prompt, narration, transition, imageURL, audioURL, shotStatus string
        err := rows.Scan(&shotID, &pID, &sceneTitle, &prompt, &narration, &transition, &imageURL, &audioURL, &shotStatus)
        if err != nil {
            log.Printf("【查询分镜】数据扫描失败 | 错误: %v", err)
            isAllCompleted = false // 扫描失败→标记为不完整
            continue
        }
        // 构造分镜数据（字段名适配前端，所有字段后加逗号）
        shotData := map[string]interface{}{
            "shotId":         shotID,
            "project_id":     pID,
            "sceneTitle":     sceneTitle,
            "prompt":         prompt,
            "narration":      narration,
            "transition":     transition,
            "localImagePath": imageURL,
            "audioPath":      audioURL,
            "status":         shotStatus,
        }
        storyboards = append(storyboards, shotData)

        // 核心判断：只要有一个分镜的核心字段为空，整体状态就是running
        if shotID == "" || imageURL == "" || audioURL == "" {
            isAllCompleted = false
        }
    }

    // 2. 处理遍历异常（有异常则标记为running）
    if err := rows.Err(); err != nil {
        log.Printf("【查询分镜】遍历数据失败 | 错误: %v", err)
        isAllCompleted = false
    }

    // 3. 确定整体status：completed/running
    var overallStatus string
    if isAllCompleted && len(storyboards) > 0 {
        overallStatus = "completed" // 所有分镜完整且有数据
    } else {
        overallStatus = "running"   // 有分镜不完整/无数据/有异常
    }

    // 4. 构造前端期望的响应格式（所有字段后加逗号）
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": gin.H{
            "status": "success",
            "project_data": gin.H{
                "storyboards": storyboards,
                "overall_status": overallStatus,
            },
        },
        "msg":  "分镜列表查询成功",
    })
}
