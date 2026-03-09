<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Http;
use Symfony\Component\Finder\Finder;

/**
 * Comprehensive knowledge ingestion into OpenBrain.
 *
 * Discovers markdown files across multiple source types and ingests
 * them as sectioned memories with embedded vectors. Designed to
 * archive scattered knowledge before filesystem cleanup.
 */
class BrainIngestCommand extends Command
{
    protected $signature = 'brain:ingest
        {--workspace= : Workspace ID to import into (required)}
        {--agent=virgil : Agent ID to attribute memories to}
        {--source=all : Source type: memory, plans, claude-md, tasks, docs, wiki, all}
        {--code-path= : Root code directory (default: ~/Code)}
        {--dry-run : Preview what would be imported without storing}
        {--fresh : Clear the Qdrant collection before ingesting}';

    protected $description = 'Ingest markdown knowledge from across the filesystem into OpenBrain';

    /** @var array<string, int> */
    private array $stats = ['imported' => 0, 'skipped' => 0, 'errors' => 0];

    public function handle(BrainService $brain): int
    {
        $workspaceId = $this->option('workspace');
        if (! $workspaceId) {
            $this->error('--workspace is required.');

            return self::FAILURE;
        }

        $source = $this->option('source') ?? 'all';
        $codePath = $this->option('code-path') ?? $this->expandHome('~/Code');
        $isDryRun = (bool) $this->option('dry-run');

        $sources = $source === 'all'
            ? ['memory', 'plans', 'claude-md', 'tasks', 'docs', 'wiki']
            : [strtolower($source)];

        // Separate file-based and API-based sources
        $fileSources = array_filter($sources, fn ($s) => $s !== 'wiki');
        $apiSources = array_filter($sources, fn ($s) => $s === 'wiki');

        // Gather file-based sources
        $filesBySource = [];
        foreach ($fileSources as $src) {
            $files = match ($src) {
                'memory' => $this->discoverMemoryFiles(),
                'plans' => $this->discoverPlanFiles($codePath),
                'claude-md' => $this->discoverClaudeMdFiles($codePath),
                'tasks' => $this->discoverTaskFiles(),
                'docs' => $this->discoverDocFiles($codePath),
                default => [],
            };
            $filesBySource[$src] = $files;
            $this->info(sprintf('  [%s] %d file(s)', $src, count($files)));
        }

        // Discover wiki pages from Forge API
        $wikiPages = [];
        if (in_array('wiki', $apiSources, true)) {
            $wikiPages = $this->discoverWikiPages();
            $this->info(sprintf('  [wiki] %d page(s) across %d repo(s)', count($wikiPages), count(array_unique(array_column($wikiPages, 'repo')))));
        }

        $totalFiles = array_sum(array_map('count', $filesBySource)) + count($wikiPages);
        $this->newLine();
        $this->info("Total: {$totalFiles} item(s) to process.");

        if ($totalFiles === 0) {
            return self::SUCCESS;
        }

        if (! $isDryRun) {
            if ($this->option('fresh')) {
                $this->warn('Clearing existing collection...');
                $this->clearCollection($brain);
            }
            $brain->ensureCollection();
        }

        foreach ($filesBySource as $src => $files) {
            $this->newLine();
            $this->comment("--- {$src} ---");

            foreach ($files as $file) {
                $this->processFile($brain, $file, $src, (int) $workspaceId, $this->option('agent') ?? 'virgil', $isDryRun);
            }
        }

        if (! empty($wikiPages)) {
            $this->newLine();
            $this->comment('--- wiki ---');
            $this->processWikiPages($brain, $wikiPages, (int) $workspaceId, $this->option('agent') ?? 'virgil', $isDryRun);
        }

        $this->newLine();
        $prefix = $isDryRun ? '[DRY RUN] ' : '';
        $this->info("{$prefix}Done. Imported: {$this->stats['imported']}, Skipped: {$this->stats['skipped']}, Errors: {$this->stats['errors']}");

        return self::SUCCESS;
    }

    /**
     * Process a single file into sectioned memories.
     */
    private function processFile(BrainService $brain, string $file, string $source, int $workspaceId, string $agentId, bool $isDryRun): void
    {
        $sections = $this->parseMarkdownSections($file);
        $filename = basename($file, '.md');
        $project = $this->extractProject($file, $source);

        if (empty($sections)) {
            $this->stats['skipped']++;

            return;
        }

        foreach ($sections as $section) {
            if (trim($section['content']) === '') {
                $this->stats['skipped']++;

                continue;
            }

            $type = $this->inferType($section['heading'], $section['content'], $source);
            $tags = $this->buildTags($section['heading'], $filename, $source, $project);

            if ($isDryRun) {
                $this->line(sprintf(
                    '  %s :: %s (%s) — %d chars [%s]',
                    $filename,
                    $section['heading'],
                    $type,
                    strlen($section['content']),
                    implode(', ', $tags),
                ));
                $this->stats['imported']++;

                continue;
            }

            try {
                $text = $section['heading']."\n\n".$section['content'];

                // embeddinggemma has a 2048-token context (~4K chars).
                // Truncate oversized sections to avoid Ollama 500 errors.
                if (strlen($text) > 3800) {
                    $text = mb_substr($text, 0, 3800).'…';
                }

                $brain->remember([
                    'workspace_id' => $workspaceId,
                    'agent_id' => $agentId,
                    'type' => $type,
                    'content' => $text,
                    'tags' => $tags,
                    'project' => $project,
                    'confidence' => $this->confidenceForSource($source),
                ]);
                $this->stats['imported']++;
            } catch (\Throwable $e) {
                $this->warn("  Error: {$filename} :: {$section['heading']} — {$e->getMessage()}");
                $this->stats['errors']++;
            }
        }
    }

    // -------------------------------------------------------------------------
    // File discovery
    // -------------------------------------------------------------------------

    /** @return array<string> */
    private function discoverMemoryFiles(): array
    {
        $pattern = $this->expandHome('~/.claude/projects/*/memory/*.md');

        return glob($pattern) ?: [];
    }

    /** @return array<string> */
    private function discoverPlanFiles(string $codePath): array
    {
        $files = [];

        // ~/.claude/plans (superpowers plans)
        $claudePlans = $this->expandHome('~/.claude/plans');
        if (is_dir($claudePlans)) {
            $files = array_merge($files, $this->findMd($claudePlans));
        }

        // docs/plans across all repos in ~/Code
        if (is_dir($codePath)) {
            $finder = Finder::create()
                ->files()
                ->name('*.md')
                ->in($codePath)
                ->path('/docs\/plans\//')
                ->notPath('node_modules')
                ->notPath('vendor')
                ->sortByName();

            foreach ($finder as $file) {
                $files[] = $file->getRealPath();
            }
        }

        return $files;
    }

    /** @return array<string> */
    private function discoverClaudeMdFiles(string $codePath): array
    {
        if (! is_dir($codePath)) {
            return [];
        }

        $finder = Finder::create()
            ->files()
            ->name('CLAUDE.md')
            ->in($codePath)
            ->depth('< 4')
            ->notPath('node_modules')
            ->notPath('vendor')
            ->notPath('.claude')
            ->sortByName();

        $files = [];
        foreach ($finder as $file) {
            $files[] = $file->getRealPath();
        }

        return $files;
    }

    /** @return array<string> */
    private function discoverTaskFiles(): array
    {
        $tasksDir = $this->expandHome('~/Code/host-uk/core/tasks');
        if (! is_dir($tasksDir)) {
            return [];
        }

        $finder = Finder::create()
            ->files()
            ->name('*.md')
            ->in($tasksDir)
            ->notPath('recovered-hostuk')
            ->notPath('recovered-root')
            ->sortByName();

        $files = [];
        foreach ($finder as $file) {
            $files[] = $file->getRealPath();
        }

        return $files;
    }

    /** @return array<string> */
    private function discoverDocFiles(string $codePath): array
    {
        $files = [];

        // CorePHP framework docs (build/php + packages)
        $docRoots = [
            $codePath.'/host-uk/core-php/docs/build/php',
            $codePath.'/host-uk/core-php/docs/packages',
        ];

        foreach ($docRoots as $root) {
            if (! is_dir($root)) {
                continue;
            }

            $finder = Finder::create()
                ->files()
                ->name('*.md')
                ->in($root)
                ->sortByName();

            foreach ($finder as $file) {
                $files[] = $file->getRealPath();
            }
        }

        return $files;
    }

    // -------------------------------------------------------------------------
    // Wiki (Forge API)
    // -------------------------------------------------------------------------

    /**
     * Discover wiki pages from all repos in the Forge org.
     *
     * Returns flat array of ['repo' => name, 'title' => title, 'content' => markdown].
     *
     * @return array<array{repo: string, title: string, content: string}>
     */
    private function discoverWikiPages(): array
    {
        $baseUrl = config('upstream.gitea.url', 'https://forge.lthn.ai');
        $token = config('upstream.gitea.token');
        $org = config('upstream.gitea.org', 'core');

        if (! $token) {
            $this->warn('No Forge token — skipping wiki source.');

            return [];
        }

        // Fetch all repos in org
        $repos = [];
        $page = 1;

        do {
            $response = Http::withHeaders(['Authorization' => 'token ' . $token])
                ->timeout(15)
                ->get("{$baseUrl}/api/v1/orgs/{$org}/repos", ['page' => $page, 'limit' => 50]);

            if (! $response->successful()) {
                $this->warn('Failed to fetch repos: ' . $response->status());
                break;
            }

            $batch = $response->json();
            if (empty($batch)) {
                break;
            }

            foreach ($batch as $r) {
                $repos[] = $r['name'];
            }
            $page++;
        } while (count($batch) === 50);

        // Fetch wiki pages for each repo
        $pages = [];

        foreach ($repos as $repo) {
            $response = Http::withHeaders(['Authorization' => 'token ' . $token])
                ->timeout(10)
                ->get("{$baseUrl}/api/v1/repos/{$org}/{$repo}/wiki/pages");

            if (! $response->successful() || $response->status() === 404) {
                continue;
            }

            $wikiList = $response->json();

            if (empty($wikiList)) {
                continue;
            }

            foreach ($wikiList as $wiki) {
                $title = $wiki['title'] ?? 'Untitled';

                // Fetch full page content
                $pageResponse = Http::withHeaders(['Authorization' => 'token ' . $token])
                    ->timeout(10)
                    ->get("{$baseUrl}/api/v1/repos/{$org}/{$repo}/wiki/page/{$title}");

                if (! $pageResponse->successful()) {
                    continue;
                }

                $content = $pageResponse->json('content_base64');
                if ($content) {
                    $content = base64_decode($content, true) ?: '';
                } else {
                    $content = '';
                }

                if (trim($content) === '') {
                    continue;
                }

                $pages[] = [
                    'repo' => $repo,
                    'title' => $title,
                    'content' => $content,
                ];
            }
        }

        return $pages;
    }

    /**
     * Process wiki pages into contextual memories.
     *
     * Each page is tagged with its repo and language, typed as service
     * documentation so the PHP orchestrator can reason about Go services.
     *
     * @param array<array{repo: string, title: string, content: string}> $pages
     */
    private function processWikiPages(BrainService $brain, array $pages, int $workspaceId, string $agentId, bool $isDryRun): void
    {
        foreach ($pages as $page) {
            $sections = $this->parseMarkdownFromString($page['content'], $page['title']);
            $repo = $page['repo'];

            // Detect language from repo name
            $lang = str_starts_with($repo, 'php-') ? 'php' : (str_starts_with($repo, 'go-') || $repo === 'go' ? 'go' : 'mixed');

            foreach ($sections as $section) {
                if (trim($section['content']) === '') {
                    $this->stats['skipped']++;

                    continue;
                }

                $tags = [
                    'source:wiki',
                    'repo:' . $repo,
                    'lang:' . $lang,
                    str_replace(['-', '_'], ' ', $page['title']),
                ];

                if ($isDryRun) {
                    $this->line(sprintf(
                        '  %s/%s :: %s — %d chars [%s]',
                        $repo,
                        $page['title'],
                        $section['heading'],
                        strlen($section['content']),
                        implode(', ', $tags),
                    ));
                    $this->stats['imported']++;

                    continue;
                }

                try {
                    // Prefix with repo context so embeddings understand the service
                    $text = "[{$repo}] {$section['heading']}\n\n{$section['content']}";

                    if (strlen($text) > 3800) {
                        $text = mb_substr($text, 0, 3800) . '…';
                    }

                    $brain->remember([
                        'workspace_id' => $workspaceId,
                        'agent_id' => $agentId,
                        'type' => 'service',
                        'content' => $text,
                        'tags' => $tags,
                        'project' => $repo,
                        'confidence' => 0.8,
                    ]);
                    $this->stats['imported']++;
                } catch (\Throwable $e) {
                    $this->warn('  Error: ' . $repo . '/' . $page['title'] . ' :: ' . $section['heading'] . ' — ' . $e->getMessage());
                    $this->stats['errors']++;
                }
            }
        }
    }

    /**
     * Parse markdown sections from a string (not a file).
     *
     * @return array<array{heading: string, content: string}>
     */
    private function parseMarkdownFromString(string $content, string $fallbackHeading): array
    {
        if (trim($content) === '') {
            return [];
        }

        $sections = [];
        $lines = explode("\n", $content);
        $currentHeading = '';
        $currentContent = [];

        foreach ($lines as $line) {
            if (preg_match('/^#{1,3}\s+(.+)$/', $line, $matches)) {
                if ($currentHeading !== '' && ! empty($currentContent)) {
                    $text = trim(implode("\n", $currentContent));
                    if ($text !== '') {
                        $sections[] = ['heading' => $currentHeading, 'content' => $text];
                    }
                }
                $currentHeading = trim($matches[1]);
                $currentContent = [];
            } else {
                $currentContent[] = $line;
            }
        }

        if ($currentHeading !== '' && ! empty($currentContent)) {
            $text = trim(implode("\n", $currentContent));
            if ($text !== '') {
                $sections[] = ['heading' => $currentHeading, 'content' => $text];
            }
        }

        if (empty($sections) && trim($content) !== '') {
            $sections[] = ['heading' => $fallbackHeading, 'content' => trim($content)];
        }

        return $sections;
    }

    /** @return array<string> */
    private function findMd(string $dir): array
    {
        $files = [];
        foreach (glob("{$dir}/*.md") ?: [] as $f) {
            $files[] = $f;
        }
        // Include subdirectories (e.g. completed/)
        foreach (glob("{$dir}/*/*.md") ?: [] as $f) {
            $files[] = $f;
        }

        return $files;
    }

    // -------------------------------------------------------------------------
    // Parsing
    // -------------------------------------------------------------------------

    /** @return array<array{heading: string, content: string}> */
    private function parseMarkdownSections(string $filePath): array
    {
        $content = file_get_contents($filePath);
        if ($content === false || trim($content) === '') {
            return [];
        }

        $sections = [];
        $lines = explode("\n", $content);
        $currentHeading = '';
        $currentContent = [];

        foreach ($lines as $line) {
            if (preg_match('/^#{1,3}\s+(.+)$/', $line, $matches)) {
                if ($currentHeading !== '' && ! empty($currentContent)) {
                    $text = trim(implode("\n", $currentContent));
                    if ($text !== '') {
                        $sections[] = ['heading' => $currentHeading, 'content' => $text];
                    }
                }
                $currentHeading = trim($matches[1]);
                $currentContent = [];
            } else {
                $currentContent[] = $line;
            }
        }

        // Flush last section
        if ($currentHeading !== '' && ! empty($currentContent)) {
            $text = trim(implode("\n", $currentContent));
            if ($text !== '') {
                $sections[] = ['heading' => $currentHeading, 'content' => $text];
            }
        }

        // If no headings found, treat entire file as one section
        if (empty($sections) && trim($content) !== '') {
            $sections[] = [
                'heading' => basename($filePath, '.md'),
                'content' => trim($content),
            ];
        }

        return $sections;
    }

    // -------------------------------------------------------------------------
    // Metadata
    // -------------------------------------------------------------------------

    private function extractProject(string $filePath, string $source): ?string
    {
        // Memory files: ~/.claude/projects/-Users-snider-Code-{project}/memory/
        if (preg_match('/projects\/[^\/]*-([^-\/]+)\/memory\//', $filePath, $m)) {
            return $m[1];
        }

        // Code repos: ~/Code/{project}/ or ~/Code/host-uk/{project}/
        if (preg_match('#/Code/host-uk/([^/]+)/#', $filePath, $m)) {
            return $m[1];
        }
        if (preg_match('#/Code/([^/]+)/#', $filePath, $m)) {
            return $m[1];
        }

        return null;
    }

    private function inferType(string $heading, string $content, string $source): string
    {
        // Source-specific defaults
        if ($source === 'plans') {
            return 'plan';
        }
        if ($source === 'claude-md') {
            return 'convention';
        }
        if ($source === 'docs') {
            return 'documentation';
        }

        $lower = strtolower($heading.' '.$content);

        $patterns = [
            'architecture' => ['architecture', 'stack', 'infrastructure', 'layer', 'service mesh'],
            'convention' => ['convention', 'standard', 'naming', 'pattern', 'rule', 'coding'],
            'decision' => ['decision', 'chose', 'strategy', 'approach', 'domain'],
            'bug' => ['bug', 'fix', 'broken', 'error', 'issue', 'lesson'],
            'plan' => ['plan', 'todo', 'roadmap', 'milestone', 'phase', 'task'],
            'research' => ['research', 'finding', 'discovery', 'analysis', 'rfc'],
        ];

        foreach ($patterns as $type => $keywords) {
            foreach ($keywords as $keyword) {
                if (str_contains($lower, $keyword)) {
                    return $type;
                }
            }
        }

        return 'observation';
    }

    /** @return array<string> */
    private function buildTags(string $heading, string $filename, string $source, ?string $project): array
    {
        $tags = ["source:{$source}"];

        if ($project) {
            $tags[] = "project:{$project}";
        }

        if ($filename !== 'MEMORY' && $filename !== 'CLAUDE') {
            $tags[] = str_replace(['-', '_'], ' ', $filename);
        }

        return $tags;
    }

    private function confidenceForSource(string $source): float
    {
        return match ($source) {
            'claude-md' => 0.9,
            'docs' => 0.85,
            'memory' => 0.8,
            'plans' => 0.6,
            'tasks' => 0.5,
            default => 0.5,
        };
    }

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    private function clearCollection(BrainService $brain): void
    {
        $reflection = new \ReflectionClass($brain);
        $prop = $reflection->getProperty('qdrantUrl');
        $qdrantUrl = $prop->getValue($brain);
        $prop = $reflection->getProperty('collection');
        $collection = $prop->getValue($brain);

        // Clear Qdrant collection.
        \Illuminate\Support\Facades\Http::withoutVerifying()
            ->timeout(10)
            ->delete("{$qdrantUrl}/collections/{$collection}");

        // Truncate the DB table so rows stay in sync with Qdrant.
        \Core\Mod\Agentic\Models\BrainMemory::query()->forceDelete();
    }

    private function expandHome(string $path): string
    {
        if (str_starts_with($path, '~/')) {
            $home = getenv('HOME') ?: ('/Users/'.get_current_user());

            return $home.substr($path, 1);
        }

        return $path;
    }
}
