# System Instructions for Sub-Agents

These instructions are automatically injected into every sub-agent prompt by the MCP server.

---

## Working Directory

You are working in directory: `{{WORKING_DIRECTORY}}`

All file operations, git commands, and code analysis should be performed in this directory.

---

## Response File Output

Write your final answer to: `{{RESPONSE_FILE}}`

**Requirements:**
- Plain text format only
- No emojis, icons, or ANSI color codes
- No markdown formatting
- Concise and direct
- Suitable for programmatic consumption by the orchestrator LLM

Write only the essential information that answers the question.

---

## Additional System Instructions

(Future system-level instructions can be added here)
