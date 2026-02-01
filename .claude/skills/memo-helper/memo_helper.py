#!/usr/bin/env python3
"""
Memo Helper - 自动登录并创建笔记到 DivineSense 服务器
"""

import json
import os
import sys
from pathlib import Path

# 默认配置
DEFAULT_SERVER = "http://39.105.209.49"
CONFIG_PATH = Path.home() / ".memo-config.json"
TOKEN_CACHE_PATH = Path.home() / ".memo-token.json"


def load_config():
    """加载配置，优先级: 配置文件 > 环境变量 > 默认值"""
    config = {
        "server": DEFAULT_SERVER,
        "username": None,
        "password": None,
    }

    # 从配置文件读取
    if CONFIG_PATH.exists():
        with open(CONFIG_PATH) as f:
            file_config = json.load(f)
            config.update(file_config)

    # 环境变量覆盖
    if os.getenv("MEMO_SERVER"):
        config["server"] = os.getenv("MEMO_SERVER")
    if os.getenv("MEMO_USERNAME"):
        config["username"] = os.getenv("MEMO_USERNAME")
    if os.getenv("MEMO_PASSWORD"):
        config["password"] = os.getenv("MEMO_PASSWORD")

    return config


def load_cached_token():
    """加载缓存的 token"""
    if TOKEN_CACHE_PATH.exists():
        try:
            with open(TOKEN_CACHE_PATH) as f:
                data = json.load(f)
                return data.get("token"), data.get("expires_at")
        except:
            pass
    return None, None


def save_token(token, expires_at):
    """保存 token 到缓存"""
    with open(TOKEN_CACHE_PATH, "w") as f:
        json.dump({"token": token, "expires_at": expires_at}, f)


def get_token(config):
    """获取有效的 access token，自动登录"""
    username = config.get("username")
    password = config.get("password")
    server = config.get("server", DEFAULT_SERVER)

    if not username or not password:
        return None, "缺少用户名或密码，请设置环境变量或配置文件"

    # 检查缓存的 token
    cached_token, expires_at = load_cached_token()
    # TODO: 验证 token 是否过期
    if cached_token:
        return cached_token, None

    # 登录获取新 token
    import subprocess
    result = subprocess.run([
        "curl", "-s", "-X", "POST",
        f"{server}/memos.api.v1.AuthService/SignIn",
        "-H", "Content-Type: application/json",
        "-d", json.dumps({
            "password_credentials": {
                "username": username,
                "password": password
            }
        })
    ], capture_output=True, text=True)

    try:
        response = json.loads(result.stdout)
        if "accessToken" in response:
            token = response["accessToken"]
            expires = response.get("accessTokenExpiresAt", "")
            save_token(token, expires)
            return token, None
        else:
            return None, response.get("message", "登录失败")
    except json.JSONDecodeError:
        return None, f"解析响应失败: {result.stdout[:200]}"


def create_memo(content, config):
    """创建笔记"""
    server = config.get("server", DEFAULT_SERVER)

    # 获取 token
    token, error = get_token(config)
    if error:
        return None, error

    # 提取标签
    tags = []
    words = content.split()
    for word in words:
        if word.startswith("#") and len(word) > 1:
            tag = word[1:].strip(".,!?;:")
            if tag:
                tags.append(tag)

    # 调用 API
    import subprocess
    payload = {
        "content": content,
        "visibility": "PRIVATE"
    }
    if tags:
        payload["tags"] = tags

    result = subprocess.run([
        "curl", "-s", "-X", "POST",
        f"{server}/api/v1/memos",
        "-H", "Content-Type: application/json",
        "-H", f"Authorization: Bearer {token}",
        "-d", json.dumps(payload)
    ], capture_output=True, text=True)

    try:
        response = json.loads(result.stdout)
        if "name" in response:
            memo_id = response["name"].split("/")[-1]
            return f"{server}/m/{memo_id}", None
        else:
            return None, response.get("message", "创建失败")
    except json.JSONDecodeError:
        return None, f"解析响应失败: {result.stdout[:200]}"


def main():
    if len(sys.argv) < 2:
        print("用法: memo_helper.py <内容>")
        print("\n配置文件:", CONFIG_PATH)
        print("环境变量: MEMO_SERVER, MEMO_USERNAME, MEMO_PASSWORD")
        sys.exit(1)

    content = " ".join(sys.argv[1:])
    config = load_config()

    url, error = create_memo(content, config)
    if error:
        print(f"错误: {error}", file=sys.stderr)
        sys.exit(1)
    else:
        print(f"✅ 笔记已创建: {url}")


if __name__ == "__main__":
    main()
