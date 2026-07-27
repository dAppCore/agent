# GitHub App Setup — dAppCore Agent

## Create the App

Go to: https://github.com/organizations/dAppCore/settings/apps/new

### Basic Info
- **App name**: `core-agent`
- **Homepage URL**: `https://core.help`
- **Description**: Automated code sync, review, and CI/CD for the Core ecosystem

### Webhook
- **Active**: Yes
- **Webhook URL**: `https://api.lthn.sh/api/github/webhook` (we'll build this endpoint)
- **Webhook secret**: (generate one — save it for the server)

### Permissions

#### Repository permissions:
- **Contents**: Read & write (push to dev branch)
- **Pull requests**: Read & write (create, merge, comment)
- **Issues**: Read & write (create from findings)
- **Checks**: Read & write (report build status)
- **Actions**: Read (check workflow status)
- **Metadata**: Read (always required)

#### Organization permissions:
- None needed

### Subscribe to events:
- Pull request
- Pull request review
- Push
- Check run
- Check suite

### Where can this app be installed?
- **Only on this account** (dAppCore org only)

## After Creation

1. Note the **App ID** and **Client ID**
2. Generate a **Private Key** (.pem file)
3. Install the app on the dAppCore organization (all repos)
4. Save credentials:
   ```bash
   mkdir -p ~/.core/github-app
   # Save the .pem file
   cp ~/Downloads/core-agent.*.pem ~/.core/github-app/private-key.pem
   # Save app ID
   echo "APP_ID" > ~/.core/github-app/app-id
   ```

## Webhook Handler

The webhook handler at `api.lthn.sh/api/github/webhook` will:

1. **pull_request_review (approved)** → auto-merge the PR
2. **pull_request_review (changes_requested)** → extract findings, dispatch fix agent
3. **push (to main)** → update Forge mirror (reverse sync)
4. **check_run (completed)** → report status back

All events are also stored in uptelligence for the CodeRabbit KPI tracking.
