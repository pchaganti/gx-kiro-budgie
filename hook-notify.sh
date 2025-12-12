#!/bin/bash

# Exit early if say command is not available
if ! command -v say &> /dev/null; then
  exit 0
fi

AGENT_NAME=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --agent-name)
      AGENT_NAME="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

#VOICE="Aman"
VOICE="Daniel"

INPUT=$(cat)

echo $INPUT

EVENT=$(echo "$INPUT" | jq -r '.hook_event_name')
CWD=$(echo "$INPUT" | jq -r '.cwd // empty')
TOOL=$(echo "$INPUT" | jq -r '.tool_name // empty')
PROMPT=$(echo "$INPUT" | jq -r '.prompt // empty')
SUCCESS=$(echo "$INPUT" | jq -r '.tool_response.success // empty')

case "$EVENT" in
  agentSpawn)
    say -v ${VOICE} "new ${AGENT_NAME} spawned!"
    ;;
  userPromptSubmit)
    WORDS=$(echo "$PROMPT" | wc -w | xargs)
    say -v ${VOICE} "prompt submitted to ${AGENT_NAME} with $WORDS words"
    ;;
  preToolUse)
    say -v ${VOICE} "${AGENT_NAME} wants to use tool $TOOL"
    ;;
  postToolUse)
    if [ "$SUCCESS" = "true" ]; then
      say -v ${VOICE} "tool $TOOL succeeded"
    else
      say -v ${VOICE} "tool $TOOL failed"
    fi
    ;;
  stop)
    say -v ${VOICE} "${AGENT_NAME} is waiting for instructions!"
    ;;
esac
