<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Console\Command;

/**
 * Import MEMORY.md files from Claude Code project memory directories
 * into the OpenBrain knowledge store.
 *
 * Scans Claude Code project memory directories (~/.claude/projects)
 * for MEMORY.md and topic-specific markdown files, parses them into
 * individual memories, and stores each via BrainService::remember().
 */
class BrainSeedMemoryCommand extends Command
{
    protected $signature = 'brain:seed-memory
        {--workspace= : Workspace ID to import into (required)}
        {--agent=virgil : Agent ID to attribute memories to}
        {--path= : Override scan path (default: ~/.claude/projects/*/memory/)}
        {--dry-run : Preview what would be imported without storing}';

    protected $description = 'Import MEMORY.md files from Claude Code project memory into OpenBrain';

    public function handle(BrainService $brain): int
    {
        $workspaceId = $this->option('workspace');
        if (! $workspaceId) {
            $this->error('--workspace is required. Pass the workspace ID to import memories into.');

            return self::FAILURE;
        }

        $agentId = $this->option('agent') ?? 'virgil';
        $isDryRun = (bool) $this->option('dry-run');

        $scanPath = $this->option('path')
            ?? $this->expandHome('~/.claude/projects/*/memory/');

        $files = $this->discoverMarkdownFiles($scanPath);
        if (empty($files)) {
            $this->info("No markdown files found in: {$scanPath}");

            return self::SUCCESS;
        }

        $this->info(sprintf('Found %d markdown file(s) to process.', count($files)));

        if (! $isDryRun) {
            $brain->ensureCollection();
        }

        $imported = 0;
        $skipped = 0;

        foreach ($files as $file) {
            $filename = basename($file, '.md');
            $project = $this->extractProject($file);
            $sections = $this->parseMarkdownSections($file);

            if (empty($sections)) {
                $this->line("  Skipped {$filename} (no sections found)");
                $skipped++;

                continue;
            }

            foreach ($sections as $section) {
                $type = $this->inferType($section['heading'], $section['content']);

                if ($isDryRun) {
                    $this->line(sprintf(
                        '  [DRY RUN] %s :: %s (%s) — %d chars',
                        $filename,
                        $section['heading'],
                        $type,
                        strlen($section['content']),
                    ));
                    $imported++;

                    continue;
                }

                try {
                    $brain->remember([
                        'workspace_id' => (int) $workspaceId,
                        'agent_id' => $agentId,
                        'type' => $type,
                        'content' => $section['heading']."\n\n".$section['content'],
                        'tags' => $this->extractTags($section['heading'], $filename),
                        'project' => $project,
                        'confidence' => 0.7,
                    ]);
                    $imported++;
                } catch (\Throwable $e) {
                    $this->warn("  Failed to import '{$section['heading']}': {$e->getMessage()}");
                    $skipped++;
                }
            }
        }

        $prefix = $isDryRun ? '[DRY RUN] ' : '';
        $this->info("{$prefix}Imported {$imported} memories, skipped {$skipped}.");

        return self::SUCCESS;
    }

    /**
     * Parse a markdown file into sections based on ## headings.
     *
     * @return array<array{heading: string, content: string}>
     */
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
                    $sections[] = [
                        'heading' => $currentHeading,
                        'content' => trim(implode("\n", $currentContent)),
                    ];
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
                $sections[] = [
                    'heading' => $currentHeading,
                    'content' => $text,
                ];
            }
        }

        return $sections;
    }

    /**
     * Extract a project name from the file path.
     *
     * Paths like ~/.claude/projects/-Users-snider-Code-eaas/memory/MEMORY.md
     * yield "eaas".
     */
    private function extractProject(string $filePath): ?string
    {
        if (preg_match('/projects\/[^\/]*-([^-\/]+)\/memory\//', $filePath, $matches)) {
            return $matches[1];
        }

        return null;
    }

    /**
     * Infer the memory type from the heading and content.
     */
    private function inferType(string $heading, string $content): string
    {
        $lower = strtolower($heading.' '.$content);

        $patterns = [
            'architecture' => ['architecture', 'stack', 'infrastructure', 'layer', 'service mesh'],
            'convention' => ['convention', 'standard', 'naming', 'pattern', 'rule', 'coding'],
            'decision' => ['decision', 'chose', 'strategy', 'approach', 'domain'],
            'bug' => ['bug', 'fix', 'broken', 'error', 'issue', 'lesson'],
            'plan' => ['plan', 'todo', 'roadmap', 'milestone', 'phase'],
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

    /**
     * Extract topic tags from the heading and filename.
     *
     * @return array<string>
     */
    private function extractTags(string $heading, string $filename): array
    {
        $tags = [];

        if ($filename !== 'MEMORY') {
            $tags[] = str_replace(['-', '_'], ' ', $filename);
        }

        $tags[] = 'memory-import';

        return $tags;
    }

    /**
     * Expand ~ to the user's home directory.
     */
    private function expandHome(string $path): string
    {
        if (str_starts_with($path, '~/')) {
            $home = getenv('HOME') ?: ('/Users/'.get_current_user());

            return $home.substr($path, 1);
        }

        return $path;
    }

    /**
     * Discover markdown files from a file path, directory, or glob.
     *
     * The command accepts a directory path like
     * `--path=/root/.claude/projects/workspace/memory/` as well as a single
     * file path like `--path=/root/.claude/projects/foo/memory/MEMORY.md`.
     *
     * @return array<string>
     */
    private function discoverMarkdownFiles(string $scanPath): array
    {
        $expandedPath = $this->expandHome($scanPath);
        if ($expandedPath === '') {
            return [];
        }

        $files = [];
        foreach ($this->expandScanTargets($expandedPath) as $target) {
            $files = array_merge($files, $this->collectMarkdownFiles($target));
        }

        $files = array_values(array_unique($files));
        sort($files);

        return $files;
    }

    /**
     * Expand a directory path or glob pattern into concrete scan targets.
     *
     * @return array<string>
     */
    private function expandScanTargets(string $scanPath): array
    {
        if ($this->hasGlobMeta($scanPath)) {
            return glob($scanPath) ?: [];
        }

        return [$scanPath];
    }

    /**
     * Collect markdown files from a file or directory.
     *
     * @return array<string>
     */
    private function collectMarkdownFiles(string $path): array
    {
        if (! file_exists($path)) {
            return [];
        }

        if (is_file($path)) {
            return $this->isMarkdownFile($path) ? [$path] : [];
        }

        if (! is_dir($path)) {
            return [];
        }

        $files = [];
        $entries = scandir($path);
        if ($entries === false) {
            return [];
        }

        foreach ($entries as $entry) {
            if ($entry === '.' || $entry === '..') {
                continue;
            }

            $childPath = $path.'/'.$entry;
            if (is_dir($childPath)) {
                $files = array_merge($files, $this->collectMarkdownFiles($childPath));

                continue;
            }

            if ($this->isMarkdownFile($childPath)) {
                $files[] = $childPath;
            }
        }

        return $files;
    }

    private function hasGlobMeta(string $path): bool
    {
        return str_contains($path, '*') || str_contains($path, '?') || str_contains($path, '[');
    }

    private function isMarkdownFile(string $path): bool
    {
        return strtolower(pathinfo($path, PATHINFO_EXTENSION)) === 'md';
    }
}
