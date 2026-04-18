<?php

namespace Core\Mod\Agentic\Mcp\Servers;

use Core\Mcp\Resources\AppConfig;
use Core\Mcp\Resources\ContentResource;
use Core\Mcp\Resources\DatabaseSchema;
use Core\Mcp\Tools\Commerce\CreateCoupon;
use Core\Mcp\Tools\Commerce\GetBillingStatus;
use Core\Mcp\Tools\Commerce\ListInvoices;
use Core\Mcp\Tools\Commerce\UpgradePlan;
use Core\Mcp\Tools\ContentTools;
use Core\Mcp\Tools\GetStats;
use Core\Mcp\Tools\ListRoutes;
use Core\Mcp\Tools\ListSites;
use Core\Mcp\Tools\ListTables;
use Core\Mcp\Tools\QueryDatabase;
use Core\Mod\Agentic\Mcp\Prompts\AnalysePerformancePrompt;
use Core\Mod\Agentic\Mcp\Prompts\ConfigureNotificationsPrompt;
use Core\Mod\Agentic\Mcp\Prompts\CreateBioPagePrompt;
use Core\Mod\Agentic\Mcp\Prompts\SetupQrCampaignPrompt;
use Laravel\Mcp\Server;
use Mod\Bio\Mcp\BioResource;

class HostHub extends Server
{
    protected string $name = 'Host Hub';

    protected string $version = '1.0.0';

    protected string $instructions = <<<'MARKDOWN'
        Host Hub MCP Server provides tools for querying and inspecting the Host UK hosting platform.

        ## System Tools
        - list-sites: List all 6 Host UK services
        - get-stats: Get current system statistics
        - list-routes: List all web routes
        - query-database: Execute read-only SQL SELECT queries
        - list-tables: List database tables

        ## Commerce Tools
        - get-billing-status: Get subscription and billing status for a workspace
        - list-invoices: List invoices for a workspace
        - create-coupon: Create a new discount coupon
        - upgrade-plan: Preview or execute a plan change

        ## Content Tools
        Manage native CMS content (blog posts, pages):
        - content_tools action=list: List content items for a workspace
        - content_tools action=read: Read full content by slug or ID
        - content_tools action=create: Create new content (draft, published, scheduled)
        - content_tools action=update: Update existing content
        - content_tools action=delete: Soft delete content
        - content_tools action=taxonomies: List categories and tags

        ## BioLink Tools (BioHost)
        Manage bio link pages, domains, pixels, themes, and notifications:

        ### Core Operations (biolink_tools)
        - biolink_tools action=list: List biolinks for a user
        - biolink_tools action=get: Get biolink details with blocks
        - biolink_tools action=create: Create new biolink page
        - biolink_tools action=update: Update biolink settings
        - biolink_tools action=delete: Delete a biolink
        - biolink_tools action=add_block: Add a block to biolink
        - biolink_tools action=update_block: Update block settings
        - biolink_tools action=delete_block: Remove a block

        ### Analytics (analytics_tools)
        - analytics_tools action=stats: Get click statistics
        - analytics_tools action=detailed: Get geo, device, referrer, UTM breakdown

        ### Domains (domain_tools)
        - domain_tools action=list: List custom domains
        - domain_tools action=add: Add domain with verification instructions
        - domain_tools action=verify: Trigger DNS verification
        - domain_tools action=delete: Remove a domain

        ### Projects (project_tools)
        - project_tools action=list: List projects
        - project_tools action=create: Create a project
        - project_tools action=update: Update a project
        - project_tools action=delete: Delete a project
        - project_tools action=move_biolink: Move biolink to project

        ### Tracking Pixels (pixel_tools)
        - pixel_tools action=list: List tracking pixels
        - pixel_tools action=create: Create pixel (Facebook, GA4, GTM, etc.)
        - pixel_tools action=update: Update pixel
        - pixel_tools action=delete: Delete pixel
        - pixel_tools action=attach: Attach pixel to biolink
        - pixel_tools action=detach: Remove pixel from biolink

        ### QR Codes (qr_tools)
        - qr_tools action=generate: Generate QR code with custom styling

        ### Themes (theme_tools)
        - theme_tools action=list: List available themes
        - theme_tools action=apply: Apply theme to biolink
        - theme_tools action=create_custom: Create custom theme
        - theme_tools action=delete: Delete custom theme
        - theme_tools action=search: Search themes
        - theme_tools action=toggle_favourite: Toggle favourite theme

        ### Social Proof (TrustHost - trust_tools)
        Manage social proof widgets and campaigns:
        - trust_campaign_tools action=list: List campaigns
        - trust_campaign_tools action=get: Get campaign details
        - trust_notification_tools action=list: List widgets for campaign
        - trust_notification_tools action=get: Get widget details
        - trust_notification_tools action=create: Create new widget
        - trust_notification_tools action=types: List available widget types
        - trust_analytics_tools action=stats: Get performance statistics

        ## Utility Tools (utility_tools)
        Execute developer utility tools (hash generators, text converters, formatters, network lookups):
        - utility_tools action=list: List all available tools
        - utility_tools action=categories: List tools grouped by category
        - utility_tools action=info tool=<slug>: Get detailed tool information
        - utility_tools action=execute tool=<slug> input={...}: Execute a tool

        Available tool categories: Marketing, Development, Design, Security, Network, Text, Converters, Generators, Link Generators, Miscellaneous

        ## Available Prompts
        - create_biolink_page: Step-by-step biolink page creation
        - setup_qr_campaign: Create QR code campaign with tracking
        - configure_notifications: Set up notification handlers
        - analyse_performance: Analyse biolink performance with recommendations

        ## Available Resources
        - config://app: Application configuration
        - schema://database: Full database schema
        - content://{workspace}/{slug}: Content item as markdown
        - biolink://{workspace}/{slug}: Biolink page as markdown
    MARKDOWN;

    protected array $tools = [
        ListSites::class,
        GetStats::class,
        ListRoutes::class,
        QueryDatabase::class,
        ListTables::class,
        // Commerce tools
        GetBillingStatus::class,
        ListInvoices::class,
        CreateCoupon::class,
        UpgradePlan::class,
        // Content tools
        ContentTools::class,
        // BioHost tools
        \Mod\Bio\Mcp\Tools\BioLinkTools::class,
        \Mod\Bio\Mcp\Tools\AnalyticsTools::class,
        \Mod\Bio\Mcp\Tools\DomainTools::class,
        \Mod\Bio\Mcp\Tools\ProjectTools::class,
        \Mod\Bio\Mcp\Tools\PixelTools::class,
        \Mod\Bio\Mcp\Tools\QrTools::class,
        \Mod\Bio\Mcp\Tools\ThemeTools::class,
        \Mod\Bio\Mcp\Tools\NotificationTools::class,
        \Mod\Bio\Mcp\Tools\SubmissionTools::class,
        \Mod\Bio\Mcp\Tools\TemplateTools::class,
        \Mod\Bio\Mcp\Tools\StaticPageTools::class,
        \Mod\Bio\Mcp\Tools\PwaTools::class,
        // TrustHost tools
        \Mod\Trust\Mcp\Tools\CampaignTools::class,
        \Mod\Trust\Mcp\Tools\NotificationTools::class,
        \Mod\Trust\Mcp\Tools\AnalyticsTools::class,
        // Utility tools
        \Mod\Tools\Mcp\Tools\UtilityTools::class,
    ];

    protected array $resources = [
        AppConfig::class,
        DatabaseSchema::class,
        ContentResource::class,
        BioResource::class,
    ];

    protected array $prompts = [
        CreateBioPagePrompt::class,
        SetupQrCampaignPrompt::class,
        ConfigureNotificationsPrompt::class,
        AnalysePerformancePrompt::class,
    ];
}
