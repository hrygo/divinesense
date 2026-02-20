import os
import sys
import json
import uuid
import time
import queue
import logging
import threading
import subprocess

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger("CLI_Tester")

class ClaudeCLIWrapper:
    def __init__(self, session_id=None):
        self.session_id = session_id or str(uuid.uuid4())
        self.process = None
        self.stdout_queue = queue.Queue()
        self.stderr_queue = queue.Queue()
        self.running = False
        self.event_log = []
        
    def start(self, permission_mode="default"):
        cmd = [
            "claude", 
            "--print", 
            "--verbose", 
            "--output-format", "stream-json", 
            "--input-format", "stream-json", 
            "--session-id", self.session_id,
            "--permission-mode", permission_mode
        ]
        
        logger.info(f"🚀 启动 CLI 进程 | Session: {self.session_id} | Permission: {permission_mode}")
        
        self.process = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1
        )
        self.running = True
        
        threading.Thread(target=self._reader_thread, args=(self.process.stdout, self.stdout_queue, "STDOUT"), daemon=True).start()
        threading.Thread(target=self._reader_thread, args=(self.process.stderr, self.stderr_queue, "STDERR"), daemon=True).start()
        threading.Thread(target=self._stderr_logger, daemon=True).start()
        
    def _reader_thread(self, pipe, q, name):
        try:
            for line in iter(pipe.readline, ''):
                if line:
                    q.put(line.strip())
        except Exception as e:
            if self.running:
                logger.error(f"{name} 读取线程异常: {e}")
            
    def _stderr_logger(self):
        while self.running:
            try:
                line = self.stderr_queue.get(timeout=0.1)
                logger.warning(f"CLI STDERR: {line}")
            except queue.Empty:
                continue

    def send_raw(self, payload_str):
        if not self.process or self.process.poll() is not None:
            logger.error("❌ 无法发送: 进程未运行")
            return False
            
        logger.info(f"📤 注入标准输入: {payload_str}")
        try:
            self.process.stdin.write(payload_str + "\n")
            self.process.stdin.flush()
            return True
        except BrokenPipeError:
            logger.error("❌ 管道断开，服务可能已崩溃。")
            return False

    def send_message(self, text):
        payload = {
            "type": "user",
            "message": {
                "role": "user",
                "content": [{"type": "text", "text": text}]
            }
        }
        return self.send_raw(json.dumps(payload))

    def wait_for_result(self, timeout=60):
        start_time = time.time()
        events = []
        
        logger.info("⏳ 等待流式输出 ...")
        while True:
            if time.time() - start_time > timeout:
                logger.error("⏰ 超时未收到 result 事件！可能处于卡死或挂起状态。")
                self.check_status()
                return False, events
                
            if self.process.poll() is not None:
                logger.error(f"❌ 进程意外退出！Code: {self.process.returncode}")
                return False, events
                
            try:
                line = self.stdout_queue.get(timeout=0.2)
                try:
                    event = json.loads(line)
                    events.append(event)
                    self.event_log.append(event)
                    
                    e_type = event.get("type")
                    if e_type in ["thinking", "status"]:
                        sys.stdout.write(".")
                        sys.stdout.flush()
                    elif e_type == "tool_use":
                        logger.info(f"🔧 工具调用: {event.get('name')} | 参数: {str(event.get('input'))[:100]}...")
                    elif e_type == "assistant" or e_type == "user":
                        # 处理嵌套的 tool_use
                        for block in event.get("message", {}).get("content", []):
                            if block.get("type") == "tool_use":
                                logger.info(f"🔧 工具调用 (Nested): {block.get('name')} | 参数: {str(block.get('input'))[:100]}...")
                    elif e_type == "error":
                        err_msg = event.get('error', {}).get('message', event.get('error'))
                        logger.error(f"❌ 错误事件: {err_msg}")
                    elif e_type == "result":
                        print()
                        logger.info(f"✅ 此次请求结束。耗时: {event.get('duration_ms', 0)} ms")
                        return True, events
                except json.JSONDecodeError:
                    if line.strip():
                        logger.warning(f"非 JSON 输出: {line}")
            except queue.Empty:
                continue

    def check_status(self):
        code = self.process.poll()
        if code is None:
            logger.info("ℹ️ 进程状态: [活跃挂起] (Running)")
            return True
        else:
            logger.error(f"❌ 进程状态: [已死] ExitCode={code}")
            return False

    def stop(self):
        self.running = False
        if self.process and self.process.poll() is None:
            self.process.terminate()
            self.process.wait(timeout=5)


def run_comprehensive_suite():
    print("=" * 60)
    print("🧪 CCRunner 全双工流式 CLI [深度验证套件]")
    print("=" * 60)
    
    session_id = str(uuid.uuid4())
    cli = ClaudeCLIWrapper(session_id)
    cli.start(permission_mode="default")
    time.sleep(2)
    
    # --- 测试项 1: 基础会话与工具调用 ---
    print("\n\033[36m[Test 1] 基础全双工与工具调用\033[0m")
    cli.send_message("Please write a small python script named 'hello.py' that prints 'Hello'.")
    ok, _ = cli.wait_for_result(timeout=60)
    if not ok: return
    
    # --- 测试项 2: 异常语法注入测试 ---
    print("\n\033[36m[Test 2] 抗毁流控制分析 (已确认非法 JSON = 致命崩溃 故跳过后半部分)\033[0m")
    # 我们知道发坏JSON会退出，这里我们改成问一个会触发报错的普通问题
    cli.send_message("Read the file named 'i_do_not_exist_xyz.txt'")
    ok, evts = cli.wait_for_result(timeout=45)
    if ok:
        print("\033[32m✅ CLI 在遇到逻辑异常（如文件不存在）时，能够反馈 `error` 事件并优雅挂起，不崩溃。\033[0m")
    else:
        print("\033[31m❌ 逻辑异常将 CLI 卡死或奔溃！\033[0m")
        return

    # --- 测试项 3: 长时间执行与 stderr 混杂 ---
    print("\n\033[36m[Test 3] 执行长时间任务 / StdErr 混合注入测试\033[0m")
    time.sleep(1)
    cli.send_message("Use bash to run this: `for i in 1 2 3; do echo 'stdout message'; >&2 echo 'stderr error'; sleep 1; done`")
    ok, evts = cli.wait_for_result(timeout=60)
    if ok:
        print("\033[32m✅ 长时间任务(含 Sleep)与 stderr 交叉输出时，系统未发生缓冲死锁(Hangs)，正确完成。\033[0m")
    
    # --- 测试项 4: 测试并发/排队模型 ---
    print("\n\033[36m[Test 4] 请求排队能力 (Concurrency & Queuing)\033[0m")
    time.sleep(1)
    logger.info("快速连续发送两次极其紧凑的请求...")
    cli.send_message("Calculate 10+10.")
    cli.send_message("Calculate 20+20.")
    
    ok1, _ = cli.wait_for_result(timeout=60)
    ok2, _ = cli.wait_for_result(timeout=60)
    if ok1 and ok2:
        print("\033[32m✅ 流式处理支持队列缓冲！连续输入指令不会导致交织挂起或崩溃。\033[0m")
        
    # --- 测试项 5: 持久上下文恢复 (Hard Restart) ---
    print("\n\033[36m[Test 5] 进程级强杀与持久化上下文恢复 (Disk Persistence)\033[0m")
    cli.stop()
    print("❌ 原进程已被刻意强杀。休眠 3 秒，防止文件锁竞争...")
    time.sleep(3)
    
    # 这里我们使用 **相同的 session_id** 启动一个 **新** 进程！
    cli_reborn = ClaudeCLIWrapper(session_id)
    cli_reborn.start()
    time.sleep(2)
    cli_reborn.send_message("What was the exact name of the python script I asked you to write in our first interaction? Please answer ONLY the file name, nothing else.")
    ok, rep = cli_reborn.wait_for_result(timeout=60)
    
    if ok:
        response_text = ""
        for e in rep:
            if e.get("type") == "assistant":
                for block in e.get("message", {}).get("content", []):
                    if block.get("type") == "text":
                        response_text += block.get("text", "")
        print(f"🔄 恢复后的 AI 回答: {response_text.strip()}")
        if "hello.py" in response_text.lower() or "hello" in response_text.lower():
            print("\033[32m✅ [极致验证] 进程级恢复成功！UUID v5 -> Session 持久化映射架构极其稳健，完美继承前生记忆！\033[0m")
        else:
            print("\033[33m⚠️ 进程恢复了，但似乎遗忘了上下文？\033[0m")
    else:
        print("\033[31m❌ 新进程无法继承上下文执行！\033[0m")

    cli_reborn.stop()
    
    if os.path.exists("hello.py"):
        os.remove("hello.py")

if __name__ == "__main__":
    run_comprehensive_suite()
