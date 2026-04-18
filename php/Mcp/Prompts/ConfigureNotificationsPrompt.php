<?php

namespace Core\Mod\Agentic\Mcp\Prompts;

use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Prompt;
use Laravel\Mcp\Server\Prompts\Argument;

/**
 * MCP prompt for configuring biolink notifications.
 *
 * Guides through setting up notification handlers for various events
 * like clicks, form submissions, and payments.
 *
 * Part of TASK-011 Phase 12: MCP Tools Expansion for BioHost (AC53).
 */
class ConfigureNotificationsPrompt extends Prompt
{
    protected string $name = 'configure_notifications';

    protected string $title = 'Configure Notifications';

    protected string $description = 'Set up notification handlers for biolink events (clicks, form submissions, etc.)';

    /**
     * @return array<int, Argument>
     */
    public function arguments(): array
    {
        return [
            new Argument(
                name: 'biolink_id',
                description: 'The ID of the biolink to configure notifications for',
                required: true
            ),
            new Argument(
                name: 'notification_type',
                description: 'Type of notification: webhook, email, slack, discord, or telegram',
                required: false
            ),
        ];
    }

    public function handle(): Response
    {
        return Response::text(<<<'PROMPT'
# Configure Biolink Notifications

Set up real-time notifications when visitors interact with your biolink page.

## Available Event Types

| Event | Description |
|-------|-------------|
| `click` | Page view or link click |
| `block_click` | Specific block clicked |
| `form_submit` | Email/phone/contact form submission |
| `payment` | Payment received (if applicable) |

## Available Handler Types

### 1. Webhook (Custom Integration)

Send HTTP POST requests to your own endpoint:
```json
{
  "action": "create_notification_handler",
  "biolink_id": <biolink_id>,
  "name": "My Webhook",
  "type": "webhook",
  "events": ["form_submit", "payment"],
  "settings": {
    "url": "https://your-server.com/webhook",
    "secret": "optional-hmac-secret"
  }
}
```

Webhook payload includes:
- Event type and timestamp
- Biolink and block details
- Visitor data (country, device type)
- Form data (for submissions)
- HMAC signature header if secret is set

### 2. Email Notifications

Send email alerts:
```json
{
  "action": "create_notification_handler",
  "biolink_id": <biolink_id>,
  "name": "Email Alerts",
  "type": "email",
  "events": ["form_submit"],
  "settings": {
    "recipients": ["alerts@example.com", "team@example.com"],
    "subject_prefix": "[BioLink]"
  }
}
```

### 3. Slack Integration

Post to a Slack channel:
```json
{
  "action": "create_notification_handler",
  "biolink_id": <biolink_id>,
  "name": "Slack Notifications",
  "type": "slack",
  "events": ["form_submit", "click"],
  "settings": {
    "webhook_url": "https://hooks.slack.com/services/T.../B.../xxx",
    "channel": "#leads",
    "username": "BioLink Bot"
  }
}
```

To get a Slack webhook URL:
1. Go to https://api.slack.com/apps
2. Create or select an app
3. Enable "Incoming Webhooks"
4. Add a webhook to your workspace

### 4. Discord Integration

Post to a Discord channel:
```json
{
  "action": "create_notification_handler",
  "biolink_id": <biolink_id>,
  "name": "Discord Notifications",
  "type": "discord",
  "events": ["form_submit"],
  "settings": {
    "webhook_url": "https://discord.com/api/webhooks/xxx/yyy",
    "username": "BioLink"
  }
}
```

To get a Discord webhook URL:
1. Open channel settings
2. Go to Integrations > Webhooks
3. Create a new webhook

### 5. Telegram Integration

Send messages to a Telegram chat:
```json
{
  "action": "create_notification_handler",
  "biolink_id": <biolink_id>,
  "name": "Telegram Alerts",
  "type": "telegram",
  "events": ["form_submit"],
  "settings": {
    "bot_token": "123456:ABC-DEF...",
    "chat_id": "-1001234567890"
  }
}
```

To set up Telegram:
1. Message @BotFather to create a bot
2. Get the bot token
3. Add the bot to your group/channel
4. Get the chat ID (use @userinfobot or API)

## Managing Handlers

### List Existing Handlers
```json
{
  "action": "list_notification_handlers",
  "biolink_id": <biolink_id>
}
```

### Update a Handler
```json
{
  "action": "update_notification_handler",
  "handler_id": <handler_id>,
  "events": ["form_submit"],
  "is_enabled": true
}
```

### Test a Handler
```json
{
  "action": "test_notification_handler",
  "handler_id": <handler_id>
}
```

### Disable or Delete
```json
{
  "action": "update_notification_handler",
  "handler_id": <handler_id>,
  "is_enabled": false
}
```

```json
{
  "action": "delete_notification_handler",
  "handler_id": <handler_id>
}
```

## Auto-Disable Behaviour

Handlers are automatically disabled after 5 consecutive failures. To re-enable:
```json
{
  "action": "update_notification_handler",
  "handler_id": <handler_id>,
  "is_enabled": true
}
```

This resets the failure counter.

---

**Tips:**
- Use form_submit events for lead generation alerts
- Combine multiple handlers for redundancy
- Test handlers after creation to verify configuration
- Monitor trigger_count and consecutive_failures in list output
PROMPT
        );
    }
}
