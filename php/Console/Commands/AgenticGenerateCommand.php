<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

class AgenticGenerateCommand extends GenerateCommand
{
    protected $signature = 'agentic:generate
        {action=status : Action: status, brief, batch, plan, queue-stats}
        {--id= : Brief or Plan ID}
        {--type=help_article : Content type: help_article, blog_post, landing_page, social_post}
        {--title= : Content title}
        {--service= : Service context (e.g., BioHost, QRHost)}
        {--keywords= : Comma-separated keywords}
        {--words=800 : Target word count}
        {--mode=full : Generation mode: draft, refine, full}
        {--sync : Run synchronously instead of queuing}
        {--limit=5 : Batch limit}
        {--priority=normal : Priority: low, normal, high, urgent}';
}
