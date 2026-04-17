<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Facades\Http;

/**
 * Prepare an agent workspace with KB, specs, TODO, and vector context.
 *
 * Automates the "domain expert" prep that was previously manual:
 * pulls repo wiki pages, copies protocol specs, generates a task
 * file from a Forge issue, and queries the vector DB for context.
 */
class PrepWorkspaceCommand extends Command
{
    protected $signature = 'agentic:prep-workspace
        {--workspace=1 : Workspace ID}
        {--repo= : Forge repo (e.g. go-ai)}
        {--issue= : Issue number to build TODO from}
        {--org=core : Forge organisation}
        {--output= : Output directory (default: ./.core)}
        {--specs-path= : Path to specs dir (default: ~/Code/host-uk/specs)}
        {--dry-run : Preview without writing files}';

    protected $description = 'Prepare an agent workspace with wiki KB, specs, TODO, and vector context';

    private string $baseUrl;

    private string $token;

    private string $org;

    private string $outputDir;

    private bool $dryRun;

    public function handle(): int
    {
        $this->baseUrl = rtrim((string) config('upstream.gitea.url', 'https://forge.lthn.ai'), '/');
        $this->token = (string) config('upstream.gitea.token', config('agentic.forge_token', ''));
        $this->org = (string) $this->option('org');
        $this->outputDir = (string) ($this->option('output') ?? getcwd() . '/.core');
        $this->dryRun = (bool) $this->option('dry-run');

        $repo = $this->option('repo');
        $issueNumber = $this->option('issue') ? (int) $this->option('issue') : null;
        $specsPath = (string) ($this->option('specs-path') ?? $this->expandHome('~/Code/host-uk/specs'));
        $workspaceId = (int) $this->option('workspace');

        if (! $this->token) {
            $this->error('No Forge token configured. Set GITEA_TOKEN or FORGE_TOKEN in .env');

            return self::FAILURE;
        }

        if (! $repo) {
            $this->error('--repo is required (e.g. --repo=go-ai)');

            return self::FAILURE;
        }

        $this->info('Preparing workspace for ' . $this->org . '/' . $repo);
        $this->info('Output: ' . $this->outputDir);

        if ($this->dryRun) {
            $this->warn('[DRY RUN] No files will be written.');
        }

        $this->newLine();

        // Create output directory structure
        if (! $this->dryRun) {
            File::ensureDirectoryExists($this->outputDir . '/kb');
            File::ensureDirectoryExists($this->outputDir . '/specs');
        }

        // Step 1: Pull wiki pages
        $wikiCount = $this->pullWiki($repo);

        // Step 2: Copy spec files
        $specsCount = $this->copySpecs($specsPath);

        // Step 3: Generate TODO from issue
        $issueTitle = null;
        $issueBody = null;
        if ($issueNumber) {
            [$issueTitle, $issueBody] = $this->generateTodo($repo, $issueNumber);
        } else {
            $this->generateTodoSkeleton($repo);
        }

        // Step 4: Generate context from vector DB
        $contextCount = $this->generateContext($repo, $workspaceId, $issueTitle, $issueBody);

        // Summary
        $this->newLine();
        $prefix = $this->dryRun ? '[DRY RUN] ' : '';
        $this->info($prefix . 'Workspace prep complete:');
        $this->line('  Wiki pages:  ' . $wikiCount);
        $this->line('  Spec files:  ' . $specsCount);
        $this->line('  TODO:        ' . ($issueTitle ? 'from issue #' . $issueNumber : 'skeleton'));
        $this->line('  Context:     ' . $contextCount . ' memories');

        return self::SUCCESS;
    }

    /**
     * Fetch wiki pages from Forge API and write to kb/ directory.
     */
    private function pullWiki(string $repo): int
    {
        $this->info('Fetching wiki pages for ' . $this->org . '/' . $repo . '...');

        $response = Http::withHeaders(['Authorization' => 'token ' . $this->token])
            ->timeout(30)
            ->get($this->baseUrl . '/api/v1/repos/' . $this->org . '/' . $repo . '/wiki/pages');

        if (! $response->successful()) {
            if ($response->status() === 404) {
                $this->warn('  No wiki found for ' . $repo);
                if (! $this->dryRun) {
                    File::put(
                        $this->outputDir . '/kb/README.md',
                        '# No wiki found for ' . $repo . "\n\nThis repo has no wiki pages on Forge.\n"
                    );
                }

                return 0;
            }
            $this->error('  Wiki API error: ' . $response->status());

            return 0;
        }

        $pages = $response->json() ?? [];

        if (empty($pages)) {
            $this->warn('  Wiki exists but has no pages.');
            if (! $this->dryRun) {
                File::put(
                    $this->outputDir . '/kb/README.md',
                    '# No wiki found for ' . $repo . "\n\nThis repo has no wiki pages on Forge.\n"
                );
            }

            return 0;
        }

        $count = 0;
        foreach ($pages as $page) {
            $title = $page['title'] ?? 'Untitled';
            $subUrl = $page['sub_url'] ?? $title;

            if ($this->dryRun) {
                $this->line('  [would fetch] ' . $title);
                $count++;

                continue;
            }

            // Fetch individual page content using sub_url (Forgejo's internal page identifier)
            $pageResponse = Http::withHeaders(['Authorization' => 'token ' . $this->token])
                ->timeout(30)
                ->get($this->baseUrl . '/api/v1/repos/' . $this->org . '/' . $repo . '/wiki/page/' . urlencode($subUrl));

            if (! $pageResponse->successful()) {
                $this->warn('  Failed to fetch: ' . $title);

                continue;
            }

            $pageData = $pageResponse->json();
            $contentBase64 = $pageData['content_base64'] ?? '';

            if (empty($contentBase64)) {
                continue;
            }

            $content = base64_decode($contentBase64);
            $filename = preg_replace('/[^a-zA-Z0-9_\-.]/', '-', $title) . '.md';

            File::put($this->outputDir . '/kb/' . $filename, $content);
            $this->line('  ' . $title);
            $count++;
        }

        $this->info('  ' . $count . ' wiki page(s) saved to kb/');

        return $count;
    }

    /**
     * Copy protocol spec files to specs/ directory.
     */
    private function copySpecs(string $specsPath): int
    {
        $this->info('Copying spec files...');

        $specFiles = ['AGENT_CONTEXT.md', 'TASK_PROTOCOL.md'];
        $count = 0;

        foreach ($specFiles as $file) {
            $source = $specsPath . '/' . $file;

            if (! File::exists($source)) {
                $this->warn('  Not found: ' . $source);

                continue;
            }

            if ($this->dryRun) {
                $this->line('  [would copy] ' . $file);
                $count++;

                continue;
            }

            File::copy($source, $this->outputDir . '/specs/' . $file);
            $this->line('  ' . $file);
            $count++;
        }

        $this->info('  ' . $count . ' spec file(s) copied.');

        return $count;
    }

    /**
     * Fetch a Forge issue and generate TODO.md in TASK_PROTOCOL format.
     *
     * @return array{0: string|null, 1: string|null} [title, body]
     */
    private function generateTodo(string $repo, int $issueNumber): array
    {
        $this->info('Generating TODO from issue #' . $issueNumber . '...');

        $response = Http::withHeaders(['Authorization' => 'token ' . $this->token])
            ->timeout(30)
            ->get($this->baseUrl . '/api/v1/repos/' . $this->org . '/' . $repo . '/issues/' . $issueNumber);

        if (! $response->successful()) {
            $this->error('  Failed to fetch issue #' . $issueNumber . ': ' . $response->status());
            $this->generateTodoSkeleton($repo);

            return [null, null];
        }

        $issue = $response->json();
        $title = $issue['title'] ?? 'Untitled';
        $body = $issue['body'] ?? '';

        // Extract objective (first paragraph or up to 500 chars)
        $objective = $this->extractObjective($body);

        // Extract checklist items
        $checklistItems = $this->extractChecklist($body);

        $todoContent = '# TASK: ' . $title . "\n\n";
        $todoContent .= '**Status:** ready' . "\n";
        $todoContent .= '**Source:** ' . $this->baseUrl . '/' . $this->org . '/' . $repo . '/issues/' . $issueNumber . "\n";
        $todoContent .= '**Created:** ' . now()->toDateTimeString() . "\n";
        $todoContent .= '**Repo:** ' . $this->org . '/' . $repo . "\n";
        $todoContent .= "\n---\n\n";

        $todoContent .= "## Objective\n\n" . $objective . "\n";
        $todoContent .= "\n---\n\n";

        $todoContent .= "## Acceptance Criteria\n\n";
        if (! empty($checklistItems)) {
            foreach ($checklistItems as $item) {
                $todoContent .= '- [ ] ' . $item . "\n";
            }
        } else {
            $todoContent .= "_No checklist items found in issue. Agent should define acceptance criteria._\n";
        }
        $todoContent .= "\n---\n\n";

        $todoContent .= "## Implementation Checklist\n\n";
        $todoContent .= "_To be filled by the agent during planning._\n";
        $todoContent .= "\n---\n\n";

        $todoContent .= "## Notes\n\n";
        $todoContent .= "Full issue body preserved below for reference.\n\n";
        $todoContent .= "<details>\n<summary>Original Issue</summary>\n\n";
        $todoContent .= $body . "\n\n";
        $todoContent .= "</details>\n";

        if ($this->dryRun) {
            $this->line('  [would write] TODO.md from: ' . $title);
            if (! empty($checklistItems)) {
                $this->line('  Checklist items: ' . count($checklistItems));
            }
        } else {
            File::put($this->outputDir . '/TODO.md', $todoContent);
            $this->line('  TODO.md generated from: ' . $title);
        }

        return [$title, $body];
    }

    /**
     * Generate a minimal TODO.md skeleton when no issue is provided.
     */
    private function generateTodoSkeleton(string $repo): void
    {
        $content = "# TASK: [Define task]\n\n";
        $content .= '**Status:** ready' . "\n";
        $content .= '**Created:** ' . now()->toDateTimeString() . "\n";
        $content .= '**Repo:** ' . $this->org . '/' . $repo . "\n";
        $content .= "\n---\n\n";
        $content .= "## Objective\n\n_Define the objective._\n";
        $content .= "\n---\n\n";
        $content .= "## Acceptance Criteria\n\n- [ ] _Define criteria_\n";
        $content .= "\n---\n\n";
        $content .= "## Implementation Checklist\n\n_To be filled by the agent._\n";

        if ($this->dryRun) {
            $this->line('  [would write] TODO.md skeleton');
        } else {
            File::put($this->outputDir . '/TODO.md', $content);
            $this->line('  TODO.md skeleton generated (no --issue provided)');
        }
    }

    /**
     * Query BrainService for relevant context and write CONTEXT.md.
     */
    private function generateContext(string $repo, int $workspaceId, ?string $issueTitle, ?string $issueBody): int
    {
        $this->info('Querying vector DB for context...');

        try {
            $brain = app(BrainService::class);

            // Query 1: Repo-specific knowledge
            $repoResults = $brain->recall(
                'How does ' . $repo . ' work? Architecture and key interfaces.',
                10,
                ['project' => $repo],
                $workspaceId
            );

            $repoMemories = $repoResults['memories'] ?? [];
            $repoScoreMap = $repoResults['scores'] ?? [];

            // Query 2: Issue-specific context
            $issueMemories = [];
            $issueScoreMap = [];
            if ($issueTitle) {
                $query = $issueTitle . ' ' . mb_substr((string) $issueBody, 0, 500);
                $issueResults = $brain->recall($query, 5, [], $workspaceId);
                $issueMemories = $issueResults['memories'] ?? [];
                $issueScoreMap = $issueResults['scores'] ?? [];
            }

            $totalMemories = count($repoMemories) + count($issueMemories);

            $content = '# Agent Context — ' . $repo . "\n\n";
            $content .= '> Auto-generated by `agentic:prep-workspace`. Query the vector DB for more.' . "\n\n";

            $content .= "## Repo Knowledge\n\n";
            if (! empty($repoMemories)) {
                foreach ($repoMemories as $i => $memory) {
                    $memId = $memory['id'] ?? '';
                    $score = $repoScoreMap[$memId] ?? 0;
                    $memContent = $memory['content'] ?? '';
                    $memProject = $memory['project'] ?? 'unknown';
                    $memType = $memory['type'] ?? 'memory';
                    $content .= '### ' . ($i + 1) . '. ' . $memProject . ' [' . $memType . '] (score: ' . round((float) $score, 3) . ")\n\n";
                    $content .= $memContent . "\n\n";
                }
            } else {
                $content .= "_No repo-specific memories found. The vector DB may not have been seeded for this repo._\n\n";
            }

            $content .= "## Task-Relevant Context\n\n";
            if (! empty($issueMemories)) {
                foreach ($issueMemories as $i => $memory) {
                    $memId = $memory['id'] ?? '';
                    $score = $issueScoreMap[$memId] ?? 0;
                    $memContent = $memory['content'] ?? '';
                    $memProject = $memory['project'] ?? 'unknown';
                    $memType = $memory['type'] ?? 'memory';
                    $content .= '### ' . ($i + 1) . '. ' . $memProject . ' [' . $memType . '] (score: ' . round((float) $score, 3) . ")\n\n";
                    $content .= $memContent . "\n\n";
                }
            } elseif ($issueTitle) {
                $content .= "_No task-relevant memories found._\n\n";
            } else {
                $content .= "_No issue provided — skipped task-specific recall._\n\n";
            }

            if ($this->dryRun) {
                $this->line('  [would write] context.md with ' . $totalMemories . ' memories');
            } else {
                File::put($this->outputDir . '/context.md', $content);
                $this->line('  context.md generated with ' . $totalMemories . ' memories');
            }

            return $totalMemories;
        } catch (\Throwable $e) {
            $this->warn('  BrainService unavailable: ' . $e->getMessage());

            $content = '# Agent Context — ' . $repo . "\n\n";
            $content .= "> Vector DB was unavailable when this workspace was prepared.\n";
            $content .= "> Run `agentic:prep-workspace` again once Ollama/Qdrant are reachable.\n";

            if (! $this->dryRun) {
                File::put($this->outputDir . '/context.md', $content);
            }

            return 0;
        }
    }

    /**
     * Extract the first paragraph or up to 500 characters as the objective.
     */
    private function extractObjective(string $body): string
    {
        if (empty($body)) {
            return '_No description provided._';
        }

        // Find first paragraph (text before a blank line)
        $paragraphs = preg_split('/\n\s*\n/', $body, 2);
        $first = trim($paragraphs[0] ?? $body);

        if (mb_strlen($first) > 500) {
            return mb_substr($first, 0, 497) . '...';
        }

        return $first;
    }

    /**
     * Extract checklist items from markdown body.
     *
     * Matches `- [ ] text` and `- [x] text` lines.
     *
     * @return array<int, string>
     */
    private function extractChecklist(string $body): array
    {
        $items = [];

        if (preg_match_all('/- \[[ xX]\] (.+)/', $body, $matches)) {
            foreach ($matches[1] as $item) {
                $items[] = trim($item);
            }
        }

        return $items;
    }

    /**
     * Expand ~ to the user's home directory.
     */
    private function expandHome(string $path): string
    {
        if (str_starts_with($path, '~/')) {
            $home = $_SERVER['HOME'] ?? getenv('HOME') ?: '/tmp';

            return $home . substr($path, 1);
        }

        return $path;
    }
}
