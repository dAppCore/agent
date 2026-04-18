<?php

namespace Core\Mod\Agentic\Mcp\Prompts;

use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Prompt;
use Laravel\Mcp\Server\Prompts\Argument;

/**
 * MCP prompt for analysing biolink performance.
 *
 * Guides through retrieving and interpreting analytics data,
 * identifying trends, and suggesting improvements.
 *
 * Part of TASK-011 Phase 12: MCP Tools Expansion for BioHost (AC53).
 */
class AnalysePerformancePrompt extends Prompt
{
    protected string $name = 'analyse_performance';

    protected string $title = 'Analyse Bio Link Performance';

    protected string $description = 'Analyse biolink analytics and provide actionable insights for improvement';

    /**
     * @return array<int, Argument>
     */
    public function arguments(): array
    {
        return [
            new Argument(
                name: 'biolink_id',
                description: 'The ID of the biolink to analyse',
                required: true
            ),
            new Argument(
                name: 'period',
                description: 'Analysis period: 7d, 30d, 90d (default: 30d)',
                required: false
            ),
        ];
    }

    public function handle(): Response
    {
        return Response::text(<<<'PROMPT'
# Analyse Bio Link Performance

This workflow helps you analyse a biolink's performance and provide actionable recommendations.

## Step 1: Gather Analytics Data

Fetch detailed analytics:
```json
{
  "action": "get_analytics_detailed",
  "biolink_id": <biolink_id>,
  "period": "30d",
  "include": ["geo", "devices", "referrers", "utm", "blocks"]
}
```

Also get basic biolink info:
```json
{
  "action": "get",
  "biolink_id": <biolink_id>
}
```

## Step 2: Analyse the Data

Review these key metrics:

### Traffic Overview
- **Total clicks**: Overall engagement
- **Unique clicks**: Individual visitors
- **Click rate trend**: Is traffic growing or declining?

### Geographic Insights
Look at the `geo.countries` data:
- Where is traffic coming from?
- Are target markets represented?
- Any unexpected sources?

### Device Breakdown
Examine `devices` data:
- Mobile vs desktop ratio
- Browser distribution
- Operating systems

**Optimisation tip:** If mobile traffic is high (>60%), ensure blocks are mobile-friendly.

### Traffic Sources
Analyse `referrers`:
- Direct traffic (typed URL, QR codes)
- Social media sources
- Search engines
- Other websites

### UTM Campaign Performance
If using UTM tracking, review `utm`:
- Which campaigns drive traffic?
- Which sources convert best?

### Block Performance
The `blocks` data shows:
- Which links get the most clicks
- Click-through rate per block
- Underperforming content

## Step 3: Identify Issues

Common issues to look for:

### Low Click-Through Rate
If total clicks are high but block clicks are low:
- Consider reordering blocks (most important first)
- Review link text clarity
- Check if call-to-action is compelling

### High Bounce Rate
If unique clicks are close to total clicks with low block engagement:
- Page may not match visitor expectations
- Loading issues on certain devices
- Content not relevant to traffic source

### Geographic Mismatch
If traffic is from unexpected regions:
- Review where links are being shared
- Consider language/localisation
- Check for bot traffic

### Mobile Performance Issues
If mobile traffic shows different patterns:
- Test page on mobile devices
- Ensure buttons are tap-friendly
- Check image loading

## Step 4: Generate Recommendations

Based on analysis, suggest:

### Quick Wins
- Reorder blocks by popularity
- Update underperforming link text
- Add missing social platforms

### Medium-Term Improvements
- Create targeted content for top traffic sources
- Implement A/B testing for key links
- Add tracking for better attribution

### Strategic Changes
- Adjust marketing spend based on source performance
- Consider custom domains for branding
- Set up notification alerts for engagement milestones

## Step 5: Present Findings

Summarise for the user:

```markdown
## Performance Summary for [Biolink Name]

### Key Metrics (Last 30 Days)
- Total Clicks: X,XXX
- Unique Visitors: X,XXX
- Top Performing Block: [Name] (XX% of clicks)

### Traffic Sources
1. [Source 1] - XX%
2. [Source 2] - XX%
3. [Source 3] - XX%

### Geographic Distribution
- [Country 1] - XX%
- [Country 2] - XX%
- [Country 3] - XX%

### Recommendations
1. [High Priority Action]
2. [Medium Priority Action]
3. [Low Priority Action]

### Next Steps
- [Specific action item]
- Schedule follow-up analysis in [timeframe]
```

---

**Analytics Periods:**
- `7d` - Last 7 days (quick check)
- `30d` - Last 30 days (standard analysis)
- `90d` - Last 90 days (trend analysis)

**Note:** Analytics retention may be limited based on the workspace's subscription tier.

**Pro Tips:**
- Compare week-over-week for seasonal patterns
- Cross-reference with marketing calendar
- Export submission data for lead quality analysis
PROMPT
        );
    }
}
