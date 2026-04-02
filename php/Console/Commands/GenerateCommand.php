<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Models\AgentPlan;
use Illuminate\Console\Command;
use Mod\Content\Jobs\GenerateContentJob;
use Mod\Content\Models\ContentBrief;
use Mod\Content\Services\AIGatewayService;

class GenerateCommand extends Command
{
    protected $signature = 'generate
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

    protected $description = 'Generate content using AI pipeline (Gemini → Claude)';

    public function handle(): int
    {
        $action = $this->argument('action');

        return match ($action) {
            'status' => $this->showStatus(),
            'brief' => $this->generateBrief(),
            'batch' => $this->processBatch(),
            'plan' => $this->generateFromPlan(),
            'queue-stats', 'stats' => $this->showQueueStats(),
            default => $this->showHelp(),
        };
    }

    protected function showStatus(): int
    {
        $pending = ContentBrief::pending()->count();
        $queued = ContentBrief::where('status', ContentBrief::STATUS_QUEUED)->count();
        $generating = ContentBrief::where('status', ContentBrief::STATUS_GENERATING)->count();
        $review = ContentBrief::needsReview()->count();
        $published = ContentBrief::where('status', ContentBrief::STATUS_PUBLISHED)->count();
        $failed = ContentBrief::where('status', ContentBrief::STATUS_FAILED)->count();

        $gateway = app(AIGatewayService::class);

        $this->newLine();
        $this->line(' <info>Content Generation Status</info>');
        $this->newLine();

        // AI Provider status
        $geminiStatus = $gateway->isGeminiAvailable() ? '<fg=green>OK</>' : '<fg=red>Not configured</>';
        $claudeStatus = $gateway->isClaudeAvailable() ? '<fg=green>OK</>' : '<fg=red>Not configured</>';

        $this->line(" Gemini: {$geminiStatus}");
        $this->line(" Claude: {$claudeStatus}");
        $this->newLine();

        // Brief counts
        $this->line(' <comment>Content Briefs:</comment>');
        $this->line("   Pending:    {$pending}");
        $this->line("   Queued:     {$queued}");
        $this->line("   Generating: {$generating}");
        $this->line("   Review:     {$review}");
        $this->line("   Published:  {$published}");
        $this->line("   Failed:     {$failed}");
        $this->newLine();

        return 0;
    }

    protected function generateBrief(): int
    {
        $title = $this->option('title');

        if (! $title) {
            $title = $this->ask('Content title');
        }

        if (! $title) {
            $this->error('Title is required');

            return 1;
        }

        $gateway = app(AIGatewayService::class);

        if (! $gateway->isAvailable()) {
            $this->error('AI providers not configured. Set GOOGLE_AI_API_KEY and ANTHROPIC_API_KEY.');

            return 1;
        }

        // Create brief
        $brief = ContentBrief::create([
            'title' => $title,
            'slug' => \Illuminate\Support\Str::slug($title),
            'content_type' => $this->option('type'),
            'service' => $this->option('service'),
            'keywords' => $this->option('keywords')
                ? array_map('trim', explode(',', $this->option('keywords')))
                : null,
            'target_word_count' => (int) $this->option('words'),
            'status' => ContentBrief::STATUS_PENDING,
        ]);

        $this->info("Created brief #{$brief->id}: {$brief->title}");

        if ($this->option('sync')) {
            return $this->runSynchronous($brief);
        }

        // Queue for async processing
        $brief->markQueued();
        GenerateContentJob::dispatch($brief, $this->option('mode'));

        $this->comment('Queued for generation.');
        $this->line('Monitor with: php artisan agentic:generate status');

        return 0;
    }

    protected function runSynchronous(ContentBrief $brief): int
    {
        $gateway = app(AIGatewayService::class);
        $mode = $this->option('mode');

        $this->line('Generating content...');
        $this->newLine();

        try {
            $startTime = microtime(true);

            if ($mode === 'full') {
                $result = $gateway->generateAndRefine($brief);
                $draftCost = $result['draft']->estimateCost();
                $refineCost = $result['refined']->estimateCost();

                $this->info('Generation complete!');
                $this->newLine();
                $this->line(' <comment>Draft (Gemini):</comment>');
                $this->line("   Model: {$result['draft']->model}");
                $this->line("   Tokens: {$result['draft']->totalTokens()}");
                $this->line("   Cost: \${$draftCost}");
                $this->newLine();
                $this->line(' <comment>Refined (Claude):</comment>');
                $this->line("   Model: {$result['refined']->model}");
                $this->line("   Tokens: {$result['refined']->totalTokens()}");
                $this->line("   Cost: \${$refineCost}");
            } elseif ($mode === 'draft') {
                $response = $gateway->generateDraft($brief);
                $brief->markDraftComplete($response->content);

                $this->info('Draft generated!');
                $this->line("   Model: {$response->model}");
                $this->line("   Tokens: {$response->totalTokens()}");
                $this->line("   Cost: \${$response->estimateCost()}");
            } else {
                $this->error("Mode '{$mode}' requires existing draft. Use 'full' or 'draft' for new briefs.");

                return 1;
            }

            $elapsed = round(microtime(true) - $startTime, 2);
            $this->newLine();
            $this->comment("Completed in {$elapsed}s");
            $this->line("Brief status: {$brief->fresh()->status}");

        } catch (\Exception $e) {
            $this->error("Generation failed: {$e->getMessage()}");
            $brief->markFailed($e->getMessage());

            return 1;
        }

        return 0;
    }

    protected function processBatch(): int
    {
        $limit = (int) $this->option('limit');

        $briefs = ContentBrief::readyToProcess()
            ->limit($limit)
            ->get();

        if ($briefs->isEmpty()) {
            $this->info('No briefs ready for processing.');

            return 0;
        }

        $this->line("Processing {$briefs->count()} briefs...");
        $this->newLine();

        foreach ($briefs as $brief) {
            GenerateContentJob::dispatch($brief, $this->option('mode'));
            $this->line("  Queued: #{$brief->id} {$brief->title}");
        }

        $this->newLine();
        $this->info("Dispatched {$briefs->count()} jobs to content-generation queue.");

        return 0;
    }

    protected function generateFromPlan(): int
    {
        $planId = $this->option('id');

        if (! $planId) {
            $planId = $this->ask('Plan ID or slug');
        }

        $plan = AgentPlan::find($planId);
        if (! $plan) {
            $plan = AgentPlan::where('slug', $planId)->first();
        }

        if (! $plan) {
            $this->error('Plan not found');

            return 1;
        }

        $this->line("Generating content from plan: {$plan->title}");
        $this->newLine();

        // Get current phase or all phases
        $phases = $plan->agentPhases()
            ->whereIn('status', ['pending', 'in_progress'])
            ->get();

        if ($phases->isEmpty()) {
            $this->info('No phases pending. Plan may be complete.');

            return 0;
        }

        $briefsCreated = 0;
        $limit = (int) $this->option('limit');

        foreach ($phases as $phase) {
            $tasks = $phase->getTasks();

            foreach ($tasks as $index => $task) {
                if ($briefsCreated >= $limit) {
                    break 2;
                }

                $taskName = is_string($task) ? $task : ($task['name'] ?? '');
                $taskStatus = is_array($task) ? ($task['status'] ?? 'pending') : 'pending';

                if ($taskStatus === 'completed') {
                    continue;
                }

                // Create brief from task
                $brief = ContentBrief::create([
                    'title' => $taskName,
                    'slug' => \Illuminate\Support\Str::slug($taskName).'-'.time(),
                    'content_type' => $this->option('type'),
                    'service' => $this->option('service') ?? ($plan->metadata['service'] ?? null),
                    'target_word_count' => (int) $this->option('words'),
                    'status' => ContentBrief::STATUS_QUEUED,
                    'metadata' => [
                        'plan_id' => $plan->id,
                        'plan_slug' => $plan->slug,
                        'phase_id' => $phase->id,
                        'phase_order' => $phase->order,
                        'task_index' => $index,
                    ],
                ]);

                GenerateContentJob::dispatch($brief, $this->option('mode'));

                $this->line("  Queued: #{$brief->id} {$taskName}");
                $briefsCreated++;
            }
        }

        $this->newLine();
        $this->info("Created and queued {$briefsCreated} briefs from plan.");

        return 0;
    }

    protected function showQueueStats(): int
    {
        $this->newLine();
        $this->line(' <info>Queue Statistics</info>');
        $this->newLine();

        // Get stats by status
        $stats = ContentBrief::query()
            ->selectRaw('status, COUNT(*) as count')
            ->groupBy('status')
            ->pluck('count', 'status')
            ->toArray();

        foreach ($stats as $status => $count) {
            $this->line("   {$status}: {$count}");
        }

        // Recent failures
        $recentFailures = ContentBrief::where('status', ContentBrief::STATUS_FAILED)
            ->orderBy('updated_at', 'desc')
            ->limit(5)
            ->get();

        if ($recentFailures->isNotEmpty()) {
            $this->newLine();
            $this->line(' <comment>Recent Failures:</comment>');
            foreach ($recentFailures as $brief) {
                $this->line("   #{$brief->id} {$brief->title}");
                if ($brief->error_message) {
                    $this->line("      <fg=red>{$brief->error_message}</>");
                }
            }
        }

        // AI Usage summary (this month)
        $this->newLine();
        $this->line(' <comment>AI Usage (This Month):</comment>');

        $usage = \Mod\Content\Models\AIUsage::thisMonth()
            ->selectRaw('provider, SUM(input_tokens) as input, SUM(output_tokens) as output, SUM(cost_estimate) as cost')
            ->groupBy('provider')
            ->get();

        if ($usage->isEmpty()) {
            $this->line('   No usage recorded this month.');
        } else {
            foreach ($usage as $row) {
                $totalTokens = number_format($row->input + $row->output);
                $cost = number_format($row->cost, 4);
                $this->line("   {$row->provider}: {$totalTokens} tokens (\${$cost})");
            }
        }

        $this->newLine();

        return 0;
    }

    protected function showHelp(): int
    {
        $this->newLine();
        $this->line(' <info>Content Generation CLI</info>');
        $this->newLine();
        $this->line(' <comment>Usage:</comment>');
        $this->line('   php artisan agentic:generate status                 Show pipeline status');
        $this->line('   php artisan agentic:generate brief --title="Topic"   Create and queue a brief');
        $this->line('   php artisan agentic:generate brief --title="Topic" --sync  Generate immediately');
        $this->line('   php artisan agentic:generate batch --limit=10        Process queued briefs');
        $this->line('   php artisan agentic:generate plan --id=1             Generate from plan tasks');
        $this->line('   php artisan agentic:generate stats                   Show queue statistics');
        $this->newLine();
        $this->line(' <comment>Options:</comment>');
        $this->line('   --type=help_article|blog_post|landing_page|social_post');
        $this->line('   --service=BioHost|QRHost|LinkHost|etc.');
        $this->line('   --keywords="seo, keywords, here"');
        $this->line('   --words=800');
        $this->line('   --mode=draft|refine|full (default: full)');
        $this->line('   --sync           Run synchronously (wait for result)');
        $this->line('   --limit=5        Batch processing limit');
        $this->newLine();
        $this->line(' <comment>Pipeline:</comment>');
        $this->line('   1. Create brief → STATUS: pending');
        $this->line('   2. Queue job    → STATUS: queued');
        $this->line('   3. Gemini draft → STATUS: generating');
        $this->line('   4. Claude refine → STATUS: review');
        $this->line('   5. Approve      → STATUS: published');
        $this->newLine();

        return 0;
    }
}
