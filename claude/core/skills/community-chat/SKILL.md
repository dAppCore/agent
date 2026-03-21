---
name: community-chat
description: Archive Discord and Telegram community discussions
---

# Community Chat Collector

Archive Discord and Telegram community discussions.

## Challenges

| Platform | Access | Automation |
|----------|--------|------------|
| Discord | Bot token or user export | Discord.py, DiscordChatExporter |
| Telegram | User account or bot | Telethon, telegram-export |

## Tools

### Discord
- **DiscordChatExporter**: https://github.com/Tyrrrz/DiscordChatExporter
  - GUI or CLI
  - Exports to HTML, JSON, TXT, CSV
  - Requires bot token or user token

### Telegram
- **telegram-export**: https://github.com/expectocode/telegram-export
  - Python-based
  - Exports messages, media, users
  - Requires API credentials

## Manual Export

### Discord Data Request
1. User Settings → Privacy & Safety
2. Request all of my Data
3. Wait for email (can take days)
4. Download and extract

### Telegram Export
1. Desktop app → Settings → Advanced
2. Export Telegram Data
3. Select chats and data types
4. Download zip

## Usage

```bash
# Generate job list for manual processing
./generate-jobs.sh lethean > jobs.txt

# Process exported Discord data
./process-discord.sh ./discord-export/ --output=./chat-archive/

# Process exported Telegram data
./process-telegram.sh ./telegram-export/ --output=./chat-archive/
```

## Output

```
chat-archive/lethean/
├── discord/
│   ├── general/
│   │   ├── 2019.json
│   │   ├── 2020.json
│   │   └── ...
│   ├── development/
│   └── channels.json
├── telegram/
│   ├── main-group/
│   └── announcements/
└── INDEX.md
```

## Known Communities

### Lethean
- Discord: https://discord.gg/lethean
- Telegram: @labormarket (historical)

### Monero
- Multiple community discords
- IRC archives (Libera.chat)

## Notes

- Respect rate limits and ToS
- Some messages may be deleted - export doesn't get them
- Media files can be large - consider text-only first
- User privacy - consider anonymization for public archive
