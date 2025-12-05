from flask import Flask, request, jsonify
from flask_cors import CORS
import requests
import time
import threading

app = Flask(__name__)
CORS(app)  # 解决跨域问题

# 原有Go服务器的回调地址（服务器环境替换为ECS内网/公网IP，本地用39.105.112.239）
GO_SERVER_CALLBACK_BASE_URL = "http://39.105.112.239:8080/api/callback"

# 模拟AI处理延迟（秒）
AI_PROCESS_DELAY = 2

def mock_ai_process(task_type, task_id, callback_url, params):
    """模拟AI处理逻辑，延迟后回调Go服务器"""
    time.sleep(AI_PROCESS_DELAY)
    
    # 构造回调数据
    callback_data = {
        "task_id": task_id,
        "status": "completed",
        "message": f"{task_type}生成成功（假AI Server）",
    }
    
    # 补充不同类型的回调数据
    if task_type == "story":
        callback_data["story"] = [{"frame_num": 1, "content": params.get("content", "默认分镜内容")}]
        callback_url = f"{GO_SERVER_CALLBACK_BASE_URL}/story"
    elif task_type == "image":
        callback_data["image_task_id"] = task_id
        callback_data["image_url"] = "https://fake-ai-server.com/images/test.jpg"
        callback_url = f"{GO_SERVER_CALLBACK_BASE_URL}/image"
    elif task_type == "video":
        callback_data["video_task_id"] = task_id
        callback_data["video_url"] = "https://fake-ai-server.com/videos/test.mp4"
        callback_data["duration"] = 10.0
        callback_url = f"{GO_SERVER_CALLBACK_BASE_URL}/video"
    
    # 回调Go服务器
    try:
        requests.post(callback_url, json=callback_data)
        print(f"✅ 假AI Server已回调Go服务器：{callback_url}")
    except Exception as e:
        print(f"❌ 回调失败：{e}")

@app.route("/api/ai/generate/<task_type>", methods=["POST"])
def generate(task_type):
    """模拟AI生成接口（接收Go服务器/前端调用）"""
    params = request.json
    task_id = params.get("task_id", f"{task_type}_fake_{int(time.time())}")
    
    # 异步处理（避免阻塞）
    threading.Thread(
        target=mock_ai_process,
        args=(task_type, task_id, GO_SERVER_CALLBACK_BASE_URL, params)
    ).start()
    
    return jsonify({
        "code": 0,
        "msg": f"假AI Server已接收{task_type}生成请求",
        "data": {"task_id": task_id, "process_delay": AI_PROCESS_DELAY}
    })

if __name__ == "__main__":
    # 假AI Server运行在8081端口（避免与Go服务器冲突）
    app.run(host="0.0.0.0", port=8081, debug=True)
