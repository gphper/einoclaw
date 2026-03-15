import pyautogui
from datetime import datetime
import os

# 获取当前工作目录
current_dir = os.getcwd()

# 生成带时间戳的文件名
timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
filename = f"screenshot_{timestamp}.png"
save_path = os.path.join(current_dir, filename)

# 截取屏幕
screenshot = pyautogui.screenshot()

# 保存截图
screenshot.save(save_path)

print(f"截图已成功保存到: {save_path}")
