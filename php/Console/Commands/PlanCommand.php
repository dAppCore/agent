<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Services\PlanTemplateService;
use Illuminate\Console\Command;

class PlanCommand extends Command
{
    protected $signature = 'plan
        {action=list : Action: list, show, create, activate, complete, archive, templates, from-template}
        {--id= : Plan ID for show/activate/complete/archive}
        {--slug= : Plan slug for show/activate/complete/archive}
        {--title= : Plan title for create}
        {--desc= : Plan description}
        {--template= : Template slug for from-template}
        {--var=* : Variables for template (format: key=value)}
        {--status= : Filter by status: draft, active, completed, archived, all}
        {--limit=20 : Limit results}
        {--phase= : Phase number for phase operations}
        {--markdown : Output as markdown}
        {--workspace= : Workspace ID to scope queries (required for multi-tenant safety)}';

    protected $description = 'Manage agent work plans';

    protected ?int $workspaceId = null;

    public function handle(): int
    {
        // Resolve workspace from option or authenticated user
        $this->workspaceId = $this->resolveWorkspaceId();

        if ($this->workspaceId === null) {
            $this->error('Workspace context required. Use --workspace=ID or ensure user is authenticated.');

            return 1;
        }

        $action = $this->argument('action');

        return match ($action) {
            'list', 'ls' => $this->listPlans(),
            'show' => $this->showPlan(),
            'create', 'new' => $this->createPlan(),
            'activate', 'start' => $this->activatePlan(),
            'complete', 'done' => $this->completePlan(),
            'archive' => $this->archivePlan(),
            'templates', 'tpl' => $this->listTemplates(),
            'from-template', 'tpl-create' => $this->createFromTemplate(),
            'progress' => $this->showProgress(),
            'phases' => $this->showPhases(),
            'phase-complete' => $this->completePhase(),
            'phase-start' => $this->startPhase(),
            default => $this->showHelp(),
        };
    }

    /**
     * Resolve workspace ID from option or authenticated user.
     */
    protected function resolveWorkspaceId(): ?int
    {
        // Explicit workspace option takes precedence
        if ($workspaceOption = $this->option('workspace')) {
            return (int) $workspaceOption;
        }

        // Fall back to authenticated user's default workspace
        $user = auth()->user();
        if ($user && method_exists($user, 'defaultHostWorkspace')) {
            $workspace = $user->defaultHostWorkspace();

            return $workspace?->id;
        }

        return null;
    }

    protected function listPlans(): int
    {
        $query = AgentPlan::forWorkspace($this->workspaceId);

        $status = $this->option('status');
        if ($status && $status !== 'all') {
            $query->where('status', $status);
        } elseif (! $status) {
            $query->notArchived();
        }

        $plans = $query->orderByStatus()
            ->orderBy('updated_at', 'desc')
            ->limit($this->option('limit'))
            ->get();

        if ($plans->isEmpty()) {
            $this->info('No plans found.');

            return 0;
        }

        $this->newLine();

        foreach ($plans as $plan) {
            $statusBadge = match ($plan->status) {
                AgentPlan::STATUS_ACTIVE => '<fg=green>ACTIVE</>',
                AgentPlan::STATUS_DRAFT => '<fg=yellow>DRAFT</>',
                AgentPlan::STATUS_COMPLETED => '<fg=blue>DONE</>',
                AgentPlan::STATUS_ARCHIVED => '<fg=gray>ARCHIVED</>',
                default => $plan->status,
            };

            $progress = $plan->getProgress();
            $progressStr = "{$progress['completed']}/{$progress['total']}";

            $line = sprintf(
                ' %s #%d %s <comment>(%s)</comment> [%s%%]',
                $statusBadge,
                $plan->id,
                $plan->title,
                $progressStr,
                $progress['percentage']
            );

            $this->line($line);
        }

        $this->newLine();
        $active = AgentPlan::forWorkspace($this->workspaceId)->active()->count();
        $draft = AgentPlan::forWorkspace($this->workspaceId)->draft()->count();
        $this->comment(" {$active} active, {$draft} draft");

        return 0;
    }

    protected function showPlan(): int
    {
        $plan = $this->findPlan();

        if (! $plan) {
            return 1;
        }

        if ($this->option('markdown')) {
            $this->line($plan->toMarkdown());

            return 0;
        }

        $progress = $plan->getProgress();

        $this->newLine();
        $this->line(" <info>#{$plan->id}</info> {$plan->title}");
        $this->line(" Slug: {$plan->slug}");
        $this->line(" Status: {$plan->status}");
        $this->line(" Progress: {$progress['percentage']}% ({$progress['completed']}/{$progress['total']} phases)");

        if ($plan->description) {
            $this->newLine();
            $this->line(" {$plan->description}");
        }

        $this->newLine();
        $this->line(' <comment>Phases:</comment>');

        foreach ($plan->agentPhases as $phase) {
            $icon = $phase->getStatusIcon();
            $taskProgress = $phase->getTaskProgress();

            $line = sprintf(
                '   %s Phase %d: %s',
                $icon,
                $phase->order,
                $phase->name
            );

            if ($taskProgress['total'] > 0) {
                $line .= " ({$taskProgress['completed']}/{$taskProgress['total']} tasks)";
            }

            $this->line($line);
        }

        $this->newLine();
        $this->comment(" Created: {$plan->created_at->diffForHumans()}");
        $this->comment(" Updated: {$plan->updated_at->diffForHumans()}");

        return 0;
    }

    protected function createPlan(): int
    {
        $title = $this->option('title');

        if (! $title) {
            $title = $this->ask('Plan title');
        }

        if (! $title) {
            $this->error('Title is required');

            return 1;
        }

        $plan = AgentPlan::create([
            'workspace_id' => $this->workspaceId,
            'title' => $title,
            'slug' => AgentPlan::generateSlug($title),
            'description' => $this->option('desc'),
            'status' => AgentPlan::STATUS_DRAFT,
        ]);

        $this->info("Created plan #{$plan->id}: {$plan->title}");
        $this->comment("Slug: {$plan->slug}");

        return 0;
    }

    protected function activatePlan(): int
    {
        $plan = $this->findPlan();

        if (! $plan) {
            return 1;
        }

        $plan->activate();
        $this->info("Activated plan #{$plan->id}: {$plan->title}");

        return 0;
    }

    protected function completePlan(): int
    {
        $plan = $this->findPlan();

        if (! $plan) {
            return 1;
        }

        $plan->complete();
        $this->info("Completed plan #{$plan->id}: {$plan->title}");

        return 0;
    }

    protected function archivePlan(): int
    {
        $plan = $this->findPlan();

        if (! $plan) {
            return 1;
        }

        $reason = $this->ask('Archive reason (optional)');
        $plan->archive($reason);
        $this->info("Archived plan #{$plan->id}: {$plan->title}");

        return 0;
    }

    protected function listTemplates(): int
    {
        $service = app(PlanTemplateService::class);
        $templates = $service->list();

        if ($templates->isEmpty()) {
            $this->info('No templates found.');
            $this->comment('Place YAML templates in: resources/plan-templates/');

            return 0;
        }

        $this->newLine();
        $this->line(' <info>Available Templates</info>');
        $this->newLine();

        foreach ($templates as $template) {
            $vars = count($template['variables'] ?? []);
            $phases = $template['phases_count'] ?? 0;

            $this->line(sprintf(
                ' <comment>%s</comment> - %s',
                $template['slug'],
                $template['name']
            ));

            if ($template['description']) {
                $this->line("   {$template['description']}");
            }

            $this->line("   {$phases} phases, {$vars} variables [{$template['category']}]");
            $this->newLine();
        }

        return 0;
    }

    protected function createFromTemplate(): int
    {
        $templateSlug = $this->option('template');

        if (! $templateSlug) {
            $templateSlug = $this->ask('Template slug');
        }

        if (! $templateSlug) {
            $this->error('Template slug is required');

            return 1;
        }

        $service = app(PlanTemplateService::class);
        $template = $service->get($templateSlug);

        if (! $template) {
            $this->error("Template not found: {$templateSlug}");

            return 1;
        }

        // Parse variables from --var options
        $variables = [];
        foreach ($this->option('var') as $var) {
            if (str_contains($var, '=')) {
                [$key, $value] = explode('=', $var, 2);
                $variables[trim($key)] = trim($value);
            }
        }

        // Validate variables
        $validation = $service->validateVariables($templateSlug, $variables);
        if (! $validation['valid']) {
            foreach ($validation['errors'] as $error) {
                $this->error($error);
            }

            return 1;
        }

        $options = [];
        if ($title = $this->option('title')) {
            $options['title'] = $title;
        }

        $plan = $service->createPlan($templateSlug, $variables, $options);

        if (! $plan) {
            $this->error('Failed to create plan from template');

            return 1;
        }

        $this->info("Created plan #{$plan->id}: {$plan->title}");
        $this->comment("From template: {$templateSlug}");
        $this->comment("Slug: {$plan->slug}");
        $this->comment("Phases: {$plan->agentPhases->count()}");

        return 0;
    }

    protected function showProgress(): int
    {
        $plan = $this->findPlan();

        if (! $plan) {
            return 1;
        }

        $progress = $plan->getProgress();

        $this->newLine();
        $this->line(" <info>{$plan->title}</info>");
        $this->newLine();

        // Progress bar
        $barLength = 40;
        $filled = (int) round(($progress['percentage'] / 100) * $barLength);
        $empty = $barLength - $filled;

        $bar = str_repeat('=', $filled).str_repeat('-', $empty);
        $this->line(" [{$bar}] {$progress['percentage']}%");
        $this->newLine();

        $this->line(" Completed: {$progress['completed']}");
        $this->line(" In Progress: {$progress['in_progress']}");
        $this->line(" Pending: {$progress['pending']}");

        return 0;
    }

    protected function showPhases(): int
    {
        $plan = $this->findPlan();

        if (! $plan) {
            return 1;
        }

        $this->newLine();
        $this->line(" <info>Phases for: {$plan->title}</info>");
        $this->newLine();

        foreach ($plan->agentPhases as $phase) {
            $icon = $phase->getStatusIcon();
            $taskProgress = $phase->getTaskProgress();

            $this->line(sprintf(
                ' %s <comment>Phase %d:</comment> %s [%s]',
                $icon,
                $phase->order,
                $phase->name,
                $phase->status
            ));

            if ($phase->description) {
                $this->line("    {$phase->description}");
            }

            if ($taskProgress['total'] > 0) {
                $this->line("    Tasks: {$taskProgress['completed']}/{$taskProgress['total']} ({$taskProgress['percentage']}%)");
            }

            // Show remaining tasks
            $remaining = $phase->getRemainingTasks();
            if (! empty($remaining) && count($remaining) <= 5) {
                foreach ($remaining as $task) {
                    $this->line("      - {$task}");
                }
            } elseif (! empty($remaining)) {
                $this->line("      ... {$taskProgress['remaining']} tasks remaining");
            }

            $this->newLine();
        }

        return 0;
    }

    protected function startPhase(): int
    {
        $plan = $this->findPlan();

        if (! $plan) {
            return 1;
        }

        $phaseNumber = $this->option('phase');
        if (! $phaseNumber) {
            $phaseNumber = $this->ask('Phase number to start');
        }

        $phase = $plan->agentPhases()->where('order', $phaseNumber)->first();

        if (! $phase) {
            $this->error("Phase {$phaseNumber} not found");

            return 1;
        }

        if (! $phase->canStart()) {
            $blockers = $phase->checkDependencies();
            $this->error("Cannot start phase {$phaseNumber} - dependencies not met:");
            foreach ($blockers as $blocker) {
                $this->line("  - Phase {$blocker['phase_order']}: {$blocker['phase_name']} ({$blocker['status']})");
            }

            return 1;
        }

        $phase->start();
        $this->info("Started phase {$phaseNumber}: {$phase->name}");

        return 0;
    }

    protected function completePhase(): int
    {
        $plan = $this->findPlan();

        if (! $plan) {
            return 1;
        }

        $phaseNumber = $this->option('phase');
        if (! $phaseNumber) {
            $phaseNumber = $this->ask('Phase number to complete');
        }

        $phase = $plan->agentPhases()->where('order', $phaseNumber)->first();

        if (! $phase) {
            $this->error("Phase {$phaseNumber} not found");

            return 1;
        }

        $phase->complete();
        $this->info("Completed phase {$phaseNumber}: {$phase->name}");

        // Check if plan is now complete
        if ($plan->fresh()->status === AgentPlan::STATUS_COMPLETED) {
            $this->info("Plan '{$plan->title}' is now complete!");
        }

        return 0;
    }

    protected function findPlan(): ?AgentPlan
    {
        $id = $this->option('id');
        $slug = $this->option('slug');

        if (! $id && ! $slug) {
            $id = $this->ask('Plan ID or slug');
        }

        if (! $id && ! $slug) {
            $this->error('Plan ID or slug is required');

            return null;
        }

        $plan = null;

        // Always scope by workspace to prevent data leakage
        $query = AgentPlan::forWorkspace($this->workspaceId);

        if ($id) {
            $plan = (clone $query)->where('id', $id)->first();
            if (! $plan) {
                $plan = (clone $query)->where('slug', $id)->first();
            }
        }

        if (! $plan && $slug) {
            $plan = (clone $query)->where('slug', $slug)->first();
        }

        if (! $plan) {
            $this->error('Plan not found');

            return null;
        }

        return $plan;
    }

    protected function showHelp(): int
    {
        $this->newLine();
        $this->line(' <info>Plan Manager</info>');
        $this->newLine();
        $this->line(' <comment>Usage:</comment>');
        $this->line('   php artisan plan list                           List active plans');
        $this->line('   php artisan plan show --id=1                    Show plan details');
        $this->line('   php artisan plan show --slug=my-plan --markdown Export as markdown');
        $this->line('   php artisan plan create --title="My Plan"       Create a new plan');
        $this->line('   php artisan plan activate --id=1                Activate a plan');
        $this->line('   php artisan plan complete --id=1                Mark plan complete');
        $this->line('   php artisan plan archive --id=1                 Archive a plan');
        $this->newLine();
        $this->line(' <comment>Templates:</comment>');
        $this->line('   php artisan plan templates                      List available templates');
        $this->line('   php artisan plan from-template --template=help-content --var="service=BioHost"');
        $this->newLine();
        $this->line(' <comment>Phases:</comment>');
        $this->line('   php artisan plan phases --id=1                  Show all phases');
        $this->line('   php artisan plan phase-start --id=1 --phase=2   Start a phase');
        $this->line('   php artisan plan phase-complete --id=1 --phase=2  Complete a phase');
        $this->line('   php artisan plan progress --id=1                Show progress bar');
        $this->newLine();
        $this->line(' <comment>Options:</comment>');
        $this->line('   --workspace=ID   Workspace ID (required if not authenticated)');
        $this->line('   --status=draft|active|completed|archived|all');
        $this->line('   --limit=20');
        $this->newLine();

        return 0;
    }
}
