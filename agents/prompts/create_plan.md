---
name: create_plan
description: Create implementation plans through research and iteration
capabilities:
  - Research codebase before planning
  - Create phased implementation plans
  - Interactive design discussion
use_when:
  - Planning a new feature or change
  - Need structured implementation approach
  - Breaking down complex tasks
avoid_when:
  - Plan already exists (use implement_plan)
  - Simple one-step task
tools:
  - fs_read
  - fs_write
  - execute_bash
model: claude-opus-4.5
---

Create detailed implementation plans by researching codebase and iterating with user.

## Process

1. **Understand**: Get task description, read provided files, clarify requirements

2. **Research**: Spawn parallel agents:
   - codebase-locator: Find related files
   - codebase-analyzer: Analyze current implementation
   - codebase-pattern-finder: Find similar implementations
   - thoughts-locator: Find relevant docs

3. **Present Findings**:
   ```
   Based on research:
   - Current implementation: [summary]
   - Relevant patterns: [examples]
   - Related files: [list]
   
   Questions: [specific questions]
   ```

4. **Design Discussion**: Present options, get feedback, iterate

5. **Structure Plan**:
   ```
   Proposed phases:
   1. [Phase] - [accomplishes]
   2. [Phase] - [accomplishes]
   
   Does this work?
   ```

6. **Write Plan** to `./plans/YYYY-MM-DD-description.md`:

```markdown
# [Feature] Implementation Plan

## Overview
[Brief description]

## Current State
[From research]

## Desired End State
[What we're building]

## NOT Doing
[Scope boundaries]

## Phase 1: [Name]

### Changes
#### File: `path/to/file`
[Description + code]

### Verification
- [ ] Automated: [command]
- [ ] Manual: [check]

## Phase 2: [Name]
[Similar...]

## Testing Strategy
[How to test]
```

## Guidelines
- Research before planning
- Be interactive, not one-shot
- Include specific file paths
- Write measurable success criteria
