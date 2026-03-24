<?php

namespace Core\Mod\Agentic\Mcp\Prompts;

use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Prompt;
use Laravel\Mcp\Server\Prompts\Argument;

/**
 * MCP prompt for setting up a QR code campaign.
 *
 * Guides through creating a short link with QR code and tracking pixel
 * for print materials, packaging, or offline-to-online campaigns.
 *
 * Part of TASK-011 Phase 12: MCP Tools Expansion for BioHost (AC53).
 */
class SetupQrCampaignPrompt extends Prompt
{
    protected string $name = 'setup_qr_campaign';

    protected string $title = 'Set Up QR Code Campaign';

    protected string $description = 'Create a short link with QR code and tracking for print materials or offline campaigns';

    /**
     * @return array<int, Argument>
     */
    public function arguments(): array
    {
        return [
            new Argument(
                name: 'destination_url',
                description: 'The URL where the QR code should redirect to',
                required: true
            ),
            new Argument(
                name: 'campaign_name',
                description: 'A name for this campaign (e.g., "Summer Flyer 2024")',
                required: true
            ),
            new Argument(
                name: 'tracking_platform',
                description: 'Analytics platform to use (google_analytics, facebook, etc.)',
                required: false
            ),
        ];
    }

    public function handle(): Response
    {
        return Response::text(<<<'PROMPT'
# Set Up a QR Code Campaign

This workflow creates a trackable short link with a QR code for print materials, packaging, or any offline-to-online campaign.

## Step 1: Gather Campaign Details

Ask the user for:
- **Destination URL**: Where should the QR code redirect?
- **Campaign name**: For organisation (e.g., "Spring 2024 Flyers")
- **UTM parameters**: Optional tracking parameters
- **QR code style**: Colour preferences, size requirements

## Step 2: Create a Short Link

Create a redirect-type biolink:
```json
{
  "action": "create",
  "user_id": <user_id>,
  "url": "<short-slug>",
  "type": "link",
  "location_url": "<destination-url>?utm_source=qr&utm_campaign=<campaign-name>"
}
```

**Tip:** Include UTM parameters in the destination URL for better attribution in Google Analytics.

## Step 3: Set Up Tracking Pixel (Optional)

If the user wants conversion tracking, create a pixel:
```json
{
  "action": "create_pixel",
  "user_id": <user_id>,
  "type": "google_analytics",
  "pixel_id": "G-XXXXXXXXXX",
  "name": "<campaign-name> Tracking"
}
```

Available pixel types:
- `google_analytics` - GA4 measurement
- `google_tag_manager` - GTM container
- `facebook` - Meta Pixel
- `tiktok` - TikTok Pixel
- `linkedin` - LinkedIn Insight Tag
- `twitter` - Twitter Pixel

Attach the pixel to the link:
```json
{
  "action": "attach_pixel",
  "biolink_id": <biolink_id>,
  "pixel_id": <pixel_id>
}
```

## Step 4: Organise in a Project

Create or use a campaign project:
```json
{
  "action": "create_project",
  "user_id": <user_id>,
  "name": "QR Campaigns 2024",
  "color": "#6366f1"
}
```

Move the link to the project:
```json
{
  "action": "move_to_project",
  "biolink_id": <biolink_id>,
  "project_id": <project_id>
}
```

## Step 5: Generate the QR Code

Generate with default settings (black on white, 400px):
```json
{
  "action": "generate_qr",
  "biolink_id": <biolink_id>
}
```

Generate with custom styling:
```json
{
  "action": "generate_qr",
  "biolink_id": <biolink_id>,
  "size": 600,
  "foreground_colour": "#1a1a1a",
  "background_colour": "#ffffff",
  "module_style": "rounded",
  "ecc_level": "H"
}
```

**QR Code Options:**
- `size`: 100-1000 pixels (default: 400)
- `format`: "png" or "svg"
- `foreground_colour`: Hex colour for QR modules (default: #000000)
- `background_colour`: Hex colour for background (default: #ffffff)
- `module_style`: "square", "rounded", or "dots"
- `ecc_level`: Error correction - "L", "M", "Q", or "H" (higher = more resilient but denser)

The response includes a `data_uri` that can be used directly in HTML or saved as an image.

## Step 6: Set Up Notifications (Optional)

Get notified when someone scans the QR code:
```json
{
  "action": "create_notification_handler",
  "biolink_id": <biolink_id>,
  "name": "<campaign-name> Alerts",
  "type": "slack",
  "events": ["click"],
  "settings": {
    "webhook_url": "https://hooks.slack.com/services/..."
  }
}
```

## Step 7: Review and Deliver

Get the final link details:
```json
{
  "action": "get",
  "biolink_id": <biolink_id>
}
```

Provide the user with:
1. The short URL for reference
2. The QR code image (data URI or downloadable)
3. Instructions for the print designer

---

**Best Practices:**
- Use error correction level "H" for QR codes on curved surfaces or small prints
- Keep foreground/background contrast high for reliable scanning
- Test the QR code on multiple devices before printing
- Include the short URL as text near the QR code as a fallback
- Use different short links for each print run to track effectiveness
PROMPT
        );
    }
}
