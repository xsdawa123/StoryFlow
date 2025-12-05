package handler

import (
"log"
"math/rand"
"net/http"
"strconv"
"time"

"story2video_server/utils"
"github.com/gin-gonic/gin"
)

// ========== 原有结构体（不改动） ==========
type GenerateStoryRequest struct {
ProjectID string `json:"project_id" binding:"required"`
RawStory  string `json:"raw_story" binding:"required"`
Style     string `json:"style" binding:"required"`
UserID    string `json:"user_id" binding:"required"`
}

type GenerateImagesRequest struct {
ProjectID string                 `json:"project_id" binding:"required"`
UserID    string                 `json:"user_id" binding:"required"`
Shot1     map[string]interface{} `json:"shot_1" binding:"required"`
}

type GenerateAudiosRequest struct {
ProjectID string                 `json:"project_id" binding:"required"`
UserID    string                 `json:"user_id" binding:"required"`
Shot1     map[string]interface{} `json:"shot_1" binding:"required"`
}

// ========== 新增：创建故事接口相关 ==========
type StoryGenerateRequest struct {
ProjectID string `json:"project_id" binding:"required"`
RawStory  string `json:"raw_story" binding:"required"`
Style     string `json:"style" binding:"required"`
UserID    string `json:"user_id" binding:"required"`
}

// 1. 创建故事接口（POST /api/client/story/generate）
func StoryGenerate(c *gin.Context) {
var req StoryGenerateRequest
if err := c.ShouldBindJSON(&req); err != nil {
log.Printf("【创建故事】解析参数失败 | 错误: %v", err)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "参数错误: " + err.Error(),
"data": nil,
})
return
}

// 调用AI Server生成故事（复用原有逻辑）
aiReq := map[string]interface{}{
"project_id":   req.ProjectID,
"raw_story":    req.RawStory,
"style":        req.Style,
"callback_url": "http://127.0.0.1:8080/api/callback/story/result",
}
log.Printf("【创建故事】准备调用AI Server | 请求参数: %+v", aiReq)

var aiResp map[string]interface{}
err := utils.PostJSON("/api/generate/story", aiReq, &aiResp)
if err != nil {
log.Printf("【创建故事】调用AI Server失败 | 错误: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"msg":  "调用AI Server失败: " + err.Error(),
"data": nil,
})
return
}

// 写入projects表
if utils.DB != nil {
_, err := utils.DB.Exec(`
INSERT INTO projects (project_id, user_id, raw_story, style, project_status)
VALUES (?, ?, ?, ?, 'generated')
ON DUPLICATE KEY UPDATE 
raw_story = VALUES(raw_story), 
style = VALUES(style), 
project_status = VALUES(project_status),
update_time = NOW()
`, req.ProjectID, req.UserID, req.RawStory, req.Style)
if err != nil {
log.Printf("【创建故事】写入projects表失败 | 项目ID: %s | 错误: %v", req.ProjectID, err)
} else {
log.Printf("【创建故事】写入projects表成功 | 项目ID: %s", req.ProjectID)
}
}

// 写入shots表（分镜初始化）
for shotKey, shotValue := range aiResp {
if len(shotKey) >= 5 && shotKey[:5] == "shot_" {
shotMap, ok := shotValue.(map[string]interface{})
if !ok {
continue
}
sceneTitle := shotMap["scene_title"].(string)
prompt := shotMap["prompt"].(string)
narration := shotMap["narration"].(string)
transition := shotMap["transition"].(string)
shotStatus := shotMap["shot_status"].(string)

if utils.DB != nil {
_, err := utils.DB.Exec(`
INSERT INTO shots (shot_id, project_id, scene_title, prompt, narration, transition, shot_status)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE 
scene_title = VALUES(scene_title),
prompt = VALUES(prompt),
narration = VALUES(narration),
transition = VALUES(transition),
shot_status = VALUES(shot_status),
update_time = NOW()
`, shotKey[5:], req.ProjectID, sceneTitle, prompt, narration, transition, shotStatus)
if err != nil {
log.Printf("【创建故事】写入shots表失败 | 项目ID: %s | shot: %s | 错误: %v", req.ProjectID, shotKey, err)
} else {
log.Printf("【创建故事】写入shots表成功 | 项目ID: %s | shot: %s", req.ProjectID, shotKey)
}
}
}
}

// 严格对齐响应格式
c.JSON(http.StatusOK, gin.H{
"code": 0,
"msg":  "success",
"data": gin.H{
"project_id": req.ProjectID,
},
})
}

// ========== 原有接口（不改动） ==========
func GenerateStory(c *gin.Context) {
var req GenerateStoryRequest
if err := c.ShouldBindJSON(&req); err != nil {
log.Printf("【分镜生成】解析参数失败 | 错误: %v | 请求体: %+v", err, req)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

log.Printf("【分镜生成】解析参数成功 | 项目ID: %s | 故事内容: %s | 风格: %s | 用户ID: %s", req.ProjectID, req.RawStory, req.Style, req.UserID)

aiReq := map[string]interface{}{
"project_id":   req.ProjectID,
"raw_story":    req.RawStory,
"style":        req.Style,
"callback_url": "http://127.0.0.1:8080/api/callback/story/result",
}

log.Printf("【分镜生成】准备调用AI Server | 路径: /api/generate/story | 请求参数: %+v", aiReq)

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

if utils.DB != nil {
_, err := utils.DB.Exec(`
INSERT INTO projects (project_id, user_id, raw_story, style, project_status)
VALUES (?, ?, ?, ?, 'generated')
ON DUPLICATE KEY UPDATE 
raw_story = VALUES(raw_story), 
style = VALUES(style), 
project_status = VALUES(project_status),
update_time = NOW()
`, req.ProjectID, req.UserID, req.RawStory, req.Style)
if err != nil {
log.Printf("【分镜生成】写入projects表失败 | 项目ID: %s | 错误: %v", req.ProjectID, err)
} else {
log.Printf("【分镜生成】写入projects表成功 | 项目ID: %s", req.ProjectID)
}
}

for shotKey, shotValue := range aiResp {
if len(shotKey) >= 5 && shotKey[:5] == "shot_" {
shotMap, ok := shotValue.(map[string]interface{})
if !ok {
continue
}
sceneTitle := shotMap["scene_title"].(string)
prompt := shotMap["prompt"].(string)
narration := shotMap["narration"].(string)
transition := shotMap["transition"].(string)
shotStatus := shotMap["shot_status"].(string)

if utils.DB != nil {
_, err := utils.DB.Exec(`
INSERT INTO shots (shot_id, project_id, scene_title, prompt, narration, transition, shot_status)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE 
scene_title = VALUES(scene_title),
prompt = VALUES(prompt),
narration = VALUES(narration),
transition = VALUES(transition),
shot_status = VALUES(shot_status),
update_time = NOW()
`, shotKey[5:], req.ProjectID, sceneTitle, prompt, narration, transition, shotStatus)
if err != nil {
log.Printf("【分镜生成】写入shots表失败 | 项目ID: %s | shot: %s | 错误: %v", req.ProjectID, shotKey, err)
} else {
log.Printf("【分镜生成】写入shots表成功 | 项目ID: %s | shot: %s", req.ProjectID, shotKey)
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

func GenerateImages(c *gin.Context) {
var req GenerateImagesRequest
if err := c.ShouldBindJSON(&req); err != nil {
log.Printf("【图片生成】解析参数失败 | 错误: %v | 请求体: %+v", err, req)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

log.Printf("【图片生成】解析参数成功 | 项目ID: %s | 用户ID: %s | shot数量: %d", req.ProjectID, req.UserID, len(req.Shot1))

aiReq := map[string]interface{}{
"project_id":   req.ProjectID,
"user_id":      req.UserID,
"shot_1":       req.Shot1,
"callback_url": "http://127.0.0.1:8080/api/callback/image/result",
}

log.Printf("【图片生成】准备调用AI Server | 路径: /api/generate/images | 请求参数: %+v", aiReq)

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

for shotKey, shotValue := range aiResp {
if len(shotKey) >= 5 && shotKey[:5] == "shot_" {
shotMap, ok := shotValue.(map[string]interface{})
if !ok {
log.Printf("【图片生成】shot数据格式错误 | shotKey: %s | 数据: %+v", shotKey, shotValue)
continue
}

shotID, ok := shotMap["shot_id"].(string)
if !ok {
log.Printf("【图片生成】shot_id缺失 | shotKey: %s | 数据: %+v", shotKey, shotMap)
continue
}

imageURL, _ := shotMap["image_url"].(string)
shotStatus, _ := shotMap["shot_status"].(string)
prompt := req.Shot1["prompt"].(string)

if utils.DB == nil {
log.Printf("【图片生成】数据库连接为空 | 无法更新shots表 | projectID: %s | shotID: %s", req.ProjectID, shotID)
continue
}

res, err := utils.DB.Exec(`
INSERT INTO shots (shot_id, project_id, prompt, image_url, shot_status)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE 
image_url = VALUES(image_url), 
shot_status = VALUES(shot_status), 
update_time = NOW()
`, shotID, req.ProjectID, prompt, imageURL, shotStatus)

if err != nil {
log.Printf("【图片生成】更新shots表失败 | projectID: %s | shotID: %s | SQL错误: %v", req.ProjectID, shotID, err)
} else {
rowsAffected, _ := res.RowsAffected()
log.Printf("【图片生成】更新shots表成功 | projectID: %s | shotID: %s | 影响行数: %d", req.ProjectID, shotID, rowsAffected)
}
}
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": aiResp,
"msg":  "图片生成任务已完成",
})
}

func GenerateAudios(c *gin.Context) {
var req GenerateAudiosRequest
if err := c.ShouldBindJSON(&req); err != nil {
log.Printf("【音频生成】解析参数失败 | 错误: %v | 请求体: %+v", err, req)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "参数错误: " + err.Error(),
})
return
}

log.Printf("【音频生成】解析参数成功 | 项目ID: %s | 风格: %s | 用户ID: %s", req.ProjectID, req.Shot1["narration"], req.UserID)

aiReq := map[string]interface{}{
"project_id":   req.ProjectID,
"user_id":      req.UserID,
"shot_1":       req.Shot1,
"callback_url": "http://127.0.0.1:8080/api/callback/audio/result",
}

log.Printf("【音频生成】准备调用AI Server | 路径: /api/generate/audios | 请求参数: %+v", aiReq)

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

for shotKey, shotValue := range aiResp {
if len(shotKey) >= 5 && shotKey[:5] == "shot_" {
shotMap, ok := shotValue.(map[string]interface{})
if !ok {
log.Printf("【音频生成】shot数据格式错误 | shotKey: %s | 数据: %+v", shotKey, shotValue)
continue
}

shotID, ok := shotMap["shot_id"].(string)
if !ok {
log.Printf("【音频生成】shot_id缺失 | shotKey: %s | 数据: %+v", shotKey, shotMap)
continue
}

audioURL, _ := shotMap["audio_url"].(string)
shotStatus, _ := shotMap["shot_status"].(string)
narration := req.Shot1["narration"].(string)

if utils.DB == nil {
log.Printf("【音频生成】数据库连接为空 | 无法更新shots表 | projectID: %s | shotID: %s", req.ProjectID, shotID)
continue
}

res, err := utils.DB.Exec(`
INSERT INTO shots (shot_id, project_id, narration, audio_url, shot_status)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE 
audio_url = VALUES(audio_url), 
shot_status = VALUES(shot_status), 
update_time = NOW()
`, shotID, req.ProjectID, narration, audioURL, shotStatus)

if err != nil {
log.Printf("【音频生成】更新shots表失败 | projectID: %s | shotID: %s | SQL错误: %v", req.ProjectID, shotID, err)
} else {
rowsAffected, _ := res.RowsAffected()
log.Printf("【音频生成】更新shots表成功 | projectID: %s | shotID: %s | 影响行数: %d", req.ProjectID, shotID, rowsAffected)
}
}
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": aiResp,
"msg":  "音频生成任务已完成",
})
}

func ListShots(c *gin.Context) {
projectID := c.Query("project_id")
if projectID == "" {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "project_id不能为空",
})
return
}

if utils.DB == nil {
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "数据库连接异常",
})
return
}

var projectName string
projectQuery := utils.DB.QueryRow(`SELECT IFNULL(raw_story, '') FROM projects WHERE project_id = ?`, projectID)
err := projectQuery.Scan(&projectName)
if err != nil || projectName == "" {
projectName = "项目名称"
}

shotsRows, err := utils.DB.Query(`
SELECT 
IFNULL(shot_id, ''),
IFNULL(scene_title, ''),
IFNULL(prompt, ''),
IFNULL(narration, ''),
IFNULL(image_url, ''),
IFNULL(audio_url, ''),
IFNULL(shot_status, ''),
IFNULL(transition, '')
FROM shots WHERE project_id = ?
`, projectID)
if err != nil {
log.Printf("【分镜查询】查询shots表失败 | projectID: %s | 错误: %v", projectID, err)
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "查询分镜失败: " + err.Error(),
})
return
}
defer shotsRows.Close()

storyboards := make([]map[string]interface{}, 0)
for shotsRows.Next() {
var shotID, sceneTitle, prompt, narration, imageURL, audioURL, shotStatus, transition string
err := shotsRows.Scan(&shotID, &sceneTitle, &prompt, &narration, &imageURL, &audioURL, &shotStatus, &transition)
if err != nil {
log.Printf("【分镜查询】解析分镜数据失败 | 错误: %v", err)
continue
}
storyboards = append(storyboards, map[string]interface{}{
"shotId":         shotID,
"sceneTitle":     sceneTitle,
"prompt":         prompt,
"narration":      narration,
"localImagePath": imageURL,
"audioPath":      audioURL,
"status":         shotStatus,
"transition":     transition,
})
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": gin.H{
"status": "completed",
"project_data": gin.H{
"id":          projectID,
"name":        projectName,
"storyboards": storyboards,
},
},
"msg": "分镜列表查询成功",
})
}

func ListProjects(c *gin.Context) {
userID := c.Query("user_id")
if userID == "" {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"data": nil,
"msg":  "user_id不能为空",
})
return
}

if utils.DB == nil {
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "数据库连接异常",
})
return
}

rows, err := utils.DB.Query(`
SELECT project_id, user_id, IFNULL(raw_story, ''), IFNULL(style, ''), IFNULL(project_status, ''), IFNULL(create_time, ''), IFNULL(update_time, '')
FROM projects WHERE user_id=?
`, userID)
if err != nil {
log.Printf("【项目查询】查询projects表失败 | userID: %s | 错误: %v", userID, err)
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"data": nil,
"msg":  "查询项目失败: " + err.Error(),
})
return
}
defer rows.Close()

var projects []map[string]interface{}
for rows.Next() {
var projectID, userID, rawStory, style, projectStatus, createTime, updateTime string
err := rows.Scan(&projectID, &userID, &rawStory, &style, &projectStatus, &createTime, &updateTime)
if err != nil {
log.Printf("【项目查询】解析数据失败 | 错误: %v", err)
continue
}
projects = append(projects, map[string]interface{}{
"project_id":     projectID,
"user_id":        userID,
"raw_story":      rawStory,
"style":          style,
"project_status": projectStatus,
"create_time":    createTime,
"update_time":    updateTime,
})
}

c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": projects,
"msg":  "项目列表查询成功",
})
}

// ========== 新增/修改：分镜轮询+重新生成图片接口 ==========
type ImageRegenerateRequest struct {
ProjectID   string `json:"project_id" binding:"required"`
ImageTaskID string `json:"image_task_id" binding:"required"`
ImageID     string `json:"image_id" binding:"required"`
Style       string `json:"style" binding:"required"`
Prompt      string `json:"prompt" binding:"required"`
FrameNum    int    `json:"frame_num" binding:"required"`
}

var regenerateTasks = make(map[string]map[string]interface{})
var randSeed = rand.New(rand.NewSource(time.Now().UnixNano()))

// 2. 轮询分镜列表接口（GET /api/client/shots/list）
func ClientListShots(c *gin.Context) {
projectID := c.Query("project_id")
if projectID == "" {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "project_id不能为空",
"data": nil,
})
return
}

if utils.DB == nil {
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"msg":  "数据库连接异常",
"data": nil,
})
return
}

// 统计分镜数量判断状态
var shotCount int
err := utils.DB.QueryRow("SELECT COUNT(*) FROM shots WHERE project_id = ?", projectID).Scan(&shotCount)
if err != nil {
log.Printf("【Client分镜轮询】统计分镜失败 | projectID: %s | 错误: %v", projectID, err)
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"msg":  "查询失败",
"data": nil,
})
return
}

// 进行中状态（严格对齐格式：无msg）
if shotCount == 0 {
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": gin.H{
"status":   "running",
"progress": 50,
},
})
return
}

// 完成状态：查询分镜+音频路径
var projectName string
projectQuery := utils.DB.QueryRow(`SELECT IFNULL(raw_story, '') FROM projects WHERE project_id = ?`, projectID)
err = projectQuery.Scan(&projectName)
if err != nil || projectName == "" {
projectName = "项目名称"
}

shotsRows, err := utils.DB.Query(`
SELECT 
IFNULL(shot_id, ''),
IFNULL(scene_title, ''),
IFNULL(prompt, ''),
IFNULL(narration, ''),
IFNULL(image_url, ''),
IFNULL(audio_url, ''),
IFNULL(shot_status, ''),
IFNULL(transition, '')
FROM shots WHERE project_id = ?
`, projectID)
if err != nil {
log.Printf("【Client分镜轮询】查询shots表失败 | projectID: %s | 错误: %v", projectID, err)
c.JSON(http.StatusInternalServerError, gin.H{
"code": 500,
"msg":  "查询分镜失败",
"data": nil,
})
return
}
defer shotsRows.Close()

storyboards := make([]map[string]interface{}, 0)
for shotsRows.Next() {
var shotID, sceneTitle, prompt, narration, imageURL, audioURL, shotStatus, transition string
err := shotsRows.Scan(&shotID, &sceneTitle, &prompt, &narration, &imageURL, &audioURL, &shotStatus, &transition)
if err != nil {
log.Printf("【Client分镜轮询】解析分镜数据失败 | 错误: %v", err)
continue
}
storyboards = append(storyboards, map[string]interface{}{
"shotId":         shotID,
"sceneTitle":     sceneTitle,
"prompt":         prompt,
"narration":      narration,
"localImagePath": imageURL,
"audioPath":      audioURL, // 新增音频路径字段
"status":         shotStatus,
"transition":     transition,
})
}

// 完成状态响应（严格对齐格式）
c.JSON(http.StatusOK, gin.H{
"code": 0,
"msg":  "分镜查询成功",
"data": gin.H{
"status": "completed",
"project_data": gin.H{
"id":          projectID,
"name":        projectName,
"storyboards": storyboards,
},
},
})
}

// 3. 重新生成图片接口（POST /api/client/image/regenerate）
func ImageRegenerate(c *gin.Context) {
var req ImageRegenerateRequest
if err := c.ShouldBindJSON(&req); err != nil {
log.Printf("【重新生成图片】解析参数失败 | 错误: %v", err)
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "参数错误: " + err.Error(),
"data": nil,
})
return
}

// 生成任务ID
taskID := "regen_task_" + strconv.Itoa(randSeed.Intn(1000000))
regenerateTasks[taskID] = map[string]interface{}{
"status":   "running",
"image_id": req.ImageID,
"prompt":   req.Prompt,
}

// 严格对齐响应格式（无msg）
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": gin.H{
"regenerate_task_id": taskID,
},
})
}

// 4. 轮询图片生成状态接口（GET /api/client/image/result）
func ImageResult(c *gin.Context) {
taskID := c.Query("image_task_id")
if taskID == "" {
c.JSON(http.StatusBadRequest, gin.H{
"code": 400,
"msg":  "image_task_id不能为空",
"data": nil,
})
return
}

task, exists := regenerateTasks[taskID]
if !exists {
c.JSON(http.StatusNotFound, gin.H{
"code": 404,
"msg":  "任务不存在",
"data": nil,
})
return
}

// 模拟任务完成
task["status"] = "completed"
task["image_url"] = "http://xxx.com/new_image_" + task["image_id"].(string) + ".png"
regenerateTasks[taskID] = task

// 严格对齐响应格式（无msg）
c.JSON(http.StatusOK, gin.H{
"code": 0,
"data": gin.H{
"status":   task["status"],
"image_id": task["image_id"],
"image_url": task["image_url"],
},
})
}
