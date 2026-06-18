#!/usr/bin/env node
const fs = require('fs');

function normalizeHookEventName(value) {
  const raw = String(value || '');
  const normalized = raw.trim().toLowerCase().replace(/-/g, '_');
  switch (normalized) {
    case 'stop':
      return 'Stop';
    case 'notification':
      return 'Notification';
    case 'user_prompt_submit':
    case 'userpromptsubmit':
      return 'UserPromptSubmit';
    case 'pre_tool_use':
    case 'pretooluse':
      return 'PreToolUse';
    case 'post_tool_use':
    case 'posttooluse':
      return 'PostToolUse';
    case 'post_tool_use_failure':
    case 'posttoolusefailure':
      return 'PostToolUseFailure';
    case 'session_start':
    case 'sessionstart':
      return 'SessionStart';
    case 'session_end':
    case 'sessionend':
      return 'SessionEnd';
    default:
      return raw;
  }
}

function stringValue(...values) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return value;
    }
  }
  return '';
}

let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (chunk) => {
  input += chunk;
});
process.stdin.on('end', () => {
  let payload = {};
  try {
    payload = input.trim() ? JSON.parse(input) : {};
  } catch (error) {
    payload = { parse_error: String(error), raw: input };
  }

  const hookEventName = normalizeHookEventName(
    payload.hookEventName || payload.hook_event_name || process.env.GROK_HOOK_EVENT
  );
  const sessionId = stringValue(
    payload.sessionId,
    payload.session_id,
    process.env.GROK_SESSION_ID
  );
  const message = stringValue(
    payload.lastAssistantMessage,
    payload.last_assistant_message,
    payload.message,
    payload.notificationMessage,
    payload.notification_message
  );

  const event = {
    source: 'grok-status-hook',
    dmuxPaneId: process.env.DMUX_PANE_ID || '',
    tmuxPaneId: process.env.DMUX_TMUX_PANE_ID || '',
    expectedDmuxPaneId: 'dmux-1781749411085',
    expectedTmuxPaneId: '%2',
    hookEventName,
    sessionId,
    turnId: stringValue(payload.turnId, payload.turn_id, sessionId),
    lastAssistantMessage: message || null,
    transcriptPath: stringValue(payload.transcriptPath, payload.transcript_path) || null,
    cwd: stringValue(payload.cwd, payload.workspaceRoot, process.env.GROK_WORKSPACE_ROOT) || process.cwd(),
    timestamp: Date.now()
  };

  if (event.dmuxPaneId !== event.expectedDmuxPaneId) {
    process.exit(0);
  }

  try {
    fs.writeFileSync('/Users/ryan/DEV/Go/Wavelet/.dmux/worktrees/handler-model-logics-user/.grok/dmux/dmux-1781749411085.json', JSON.stringify(event, null, 2));
  } catch (error) {
    process.exit(0);
  }
});
