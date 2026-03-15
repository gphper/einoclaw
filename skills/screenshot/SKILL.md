---
name: screenshot
description: A skill to capture screenshots of current screen using a Python script.
---

# Screenshot Skill

This skill captures a screenshot of current screen and saves it as a PNG file with a timestamp.

## Capability

The skill provides a Python script named `screenshot.py` located in this directory.

## Usage

**IMPORTANT: You MUST use the `exec` tool to execute commands. Do NOT execute commands through the skill backend.**

### Steps

1. Call the `exec` tool with the following parameters:
   - command: `python {{.BaseDirectory}}/scripts/screenshot.py`

2. The screenshot will be saved in the current working directory.

### Example Tool Call

Tool: `exec`
Arguments:
```json
{
  "command": "python {{.BaseDirectory}}/scripts/screenshot.py"
}
```

**Note**: The screenshot will be saved with a filename format like `screenshot_YYYYMMDD_HHMMSS.png`.
