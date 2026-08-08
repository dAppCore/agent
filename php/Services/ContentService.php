<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Content\Models\ContentItem;
use Illuminate\Support\Facades\File;
use Symfony\Component\Yaml\Yaml;

class ContentService
{
    protected string $batchPath;

    protected string $promptPath;

    protected string $draftsPath;

    public function __construct(
        protected AgenticManager $ai
    ) {
        $this->batchPath = config('mcp.content.batch_path', 'app/Mod/Agentic/Resources/tasks');
        $this->promptPath = config('mcp.content.prompt_path', 'app/Mod/Agentic/Resources/prompts/content');
        $this->draftsPath = config('mcp.content.drafts_path', 'app/Mod/Agentic/Resources/drafts');
    }

    /**
     * Load a batch specification from markdown file.
     */
    public function loadBatch(string $batchId): ?array
    {
        $file = base_path("{$this->batchPath}/{$batchId}.md");

        if (! File::exists($file)) {
            return null;
        }

        $content = File::get($file);

        return $this->parseBatchSpec($content);
    }

    /**
     * List all available batches.
     */
    public function listBatches(): array
    {
        $files = File::glob(base_path("{$this->batchPath}/batch-*.md"));
        $batches = [];

        foreach ($files as $file) {
            $batchId = pathinfo($file, PATHINFO_FILENAME);
            $spec = $this->loadBatch($batchId);

            if ($spec) {
                $batches[] = [
                    'id' => $batchId,
                    'service' => $spec['service'] ?? 'Unknown',
                    'category' => $spec['category'] ?? 'Unknown',
                    'article_count' => count($spec['articles'] ?? []),
                    'priority' => $spec['priority'] ?? 'normal',
                ];
            }
        }

        return $batches;
    }

    /**
     * Get batch generation status.
     */
    public function getBatchStatus(string $batchId): array
    {
        $spec = $this->loadBatch($batchId);
        if (! $spec) {
            return ['error' => 'Batch not found'];
        }

        $articles = $spec['articles'] ?? [];
        $generated = 0;
        $drafted = 0;
        $published = 0;

        foreach ($articles as $article) {
            $slug = $article['slug'] ?? null;
            if (! $slug) {
                continue;
            }

            // Check if draft exists
            $draftPath = $this->getDraftPath($spec, $slug);
            if (File::exists($draftPath)) {
                $drafted++;
            }

            // Check if published in WordPress
            $item = ContentItem::where('slug', $slug)->first();
            if ($item) {
                $generated++;
                if ($item->status === 'publish') {
                    $published++;
                }
            }
        }

        return [
            'batch_id' => $batchId,
            'service' => $spec['service'] ?? 'Unknown',
            'category' => $spec['category'] ?? 'Unknown',
            'total' => count($articles),
            'drafted' => $drafted,
            'generated' => $generated,
            'published' => $published,
            'remaining' => count($articles) - $drafted,
        ];
    }

    /**
     * Generate content for a batch.
     *
     * Progress is persisted to a state file after each article so the batch
     * can be resumed after a partial failure. Call generateBatch() or
     * resumeBatch() again to pick up from the last saved state.
     *
     * @param  string  $batchId  Batch identifier (e.g., 'batch-001-link-getting-started')
     * @param  string  $provider  AI provider ('gemini' for bulk, 'claude' for refinement)
     * @param  bool  $dryRun  If true, shows what would be generated without creating files
     * @param  int  $maxRetries  Extra attempts per article on failure (0 = no retry)
     * @return array Generation results
     */
    public function generateBatch(
        string $batchId,
        string $provider = 'gemini',
        bool $dryRun = false,
        int $maxRetries = 1,
    ): array {
        $spec = $this->loadBatch($batchId);
        if (! $spec) {
            return ['error' => "Batch not found: {$batchId}"];
        }

        $results = [
            'batch_id' => $batchId,
            'provider' => $provider,
            'articles' => [],
            'generated' => 0,
            'skipped' => 0,
            'failed' => 0,
        ];

        $promptTemplate = $this->loadPromptTemplate('help-article');

        // Load or initialise progress state (skipped for dry runs)
        $progress = null;
        if (! $dryRun) {
            $progress = $this->loadBatchProgress($batchId)
                ?? $this->initialiseBatchState($batchId, $spec['articles'] ?? [], $provider);
        }

        foreach ($spec['articles'] ?? [] as $article) {
            $slug = $article['slug'] ?? null;
            if (! $slug) {
                continue;
            }

            $draftPath = $this->getDraftPath($spec, $slug);

            // Skip if draft file already exists on disk
            if (File::exists($draftPath)) {
                $results['articles'][$slug] = ['status' => 'skipped', 'reason' => 'already drafted'];
                $results['skipped']++;
                if ($progress !== null) {
                    $progress['articles'][$slug]['status'] = 'skipped';
                }

                continue;
            }

            if ($dryRun) {
                $results['articles'][$slug] = ['status' => 'would_generate', 'path' => $draftPath];

                continue;
            }

            // Skip articles successfully generated in a prior run
            if (($progress['articles'][$slug]['status'] ?? 'pending') === 'generated') {
                $results['articles'][$slug] = ['status' => 'skipped', 'reason' => 'previously generated'];
                $results['skipped']++;

                continue;
            }

            $priorAttempts = $progress['articles'][$slug]['attempts'] ?? 0;
            $articleResult = $this->attemptArticleGeneration($article, $spec, $promptTemplate, $provider, $maxRetries);

            if ($articleResult['status'] === 'generated') {
                $results['articles'][$slug] = ['status' => 'generated', 'path' => $articleResult['path']];
                $results['generated']++;
                $progress['articles'][$slug] = [
                    'status' => 'generated',
                    'attempts' => $priorAttempts + $articleResult['attempts'],
                    'last_error' => null,
                    'generated_at' => now()->toIso8601String(),
                    'last_attempt_at' => now()->toIso8601String(),
                ];
            } else {
                $results['articles'][$slug] = ['status' => 'failed', 'error' => $articleResult['error']];
                $results['failed']++;
                $progress['articles'][$slug] = [
                    'status' => 'failed',
                    'attempts' => $priorAttempts + $articleResult['attempts'],
                    'last_error' => $articleResult['error'],
                    'generated_at' => null,
                    'last_attempt_at' => now()->toIso8601String(),
                ];
            }

            // Persist after each article so a crash mid-batch is recoverable
            $progress['last_updated'] = now()->toIso8601String();
            $this->saveBatchProgress($batchId, $progress);
        }

        if ($progress !== null) {
            $progress['last_updated'] = now()->toIso8601String();
            $this->saveBatchProgress($batchId, $progress);
        }

        return $results;
    }

    /**
     * Resume a batch from its last saved state.
     *
     * Articles that were successfully generated are skipped; failed and
     * pending articles are retried. Returns an error if no progress state
     * exists (i.e. generateBatch() has never been called for this batch).
     */
    public function resumeBatch(string $batchId, ?string $provider = null, int $maxRetries = 1): array
    {
        $progress = $this->loadBatchProgress($batchId);

        if ($progress === null) {
            return ['error' => "No progress state found for batch: {$batchId}"];
        }

        $provider ??= $progress['provider'] ?? 'gemini';

        $result = $this->generateBatch($batchId, $provider, false, $maxRetries);
        $result['resumed_from'] = $progress['last_updated'];

        return $result;
    }

    /**
     * Load batch progress state from the state file.
     *
     * Returns null when no state file exists (batch has not been started).
     */
    public function loadBatchProgress(string $batchId): ?array
    {
        $path = $this->getProgressPath($batchId);

        if (! File::exists($path)) {
            return null;
        }

        $data = json_decode(File::get($path), true);

        return is_array($data) ? $data : null;
    }

    /**
     * Attempt to generate a single article with retry logic.
     *
     * Returns ['status' => 'generated', 'path' => ..., 'attempts' => N]
     *      or ['status' => 'failed',    'error' => ..., 'attempts' => N].
     */
    protected function attemptArticleGeneration(
        array $article,
        array $spec,
        string $promptTemplate,
        string $provider,
        int $maxRetries,
    ): array {
        $draftPath = $this->getDraftPath($spec, $article['slug']);
        $lastError = null;
        $totalAttempts = $maxRetries + 1;

        for ($attempt = 1; $attempt <= $totalAttempts; $attempt++) {
            try {
                $content = $this->generateArticle($article, $spec, $promptTemplate, $provider);
                $this->saveDraft($draftPath, $content, $article);

                return ['status' => 'generated', 'path' => $draftPath, 'attempts' => $attempt];
            } catch (\Exception $e) {
                $lastError = $e->getMessage();
            }
        }

        return ['status' => 'failed', 'error' => $lastError, 'attempts' => $totalAttempts];
    }

    /**
     * Initialise a fresh batch progress state.
     */
    protected function initialiseBatchState(string $batchId, array $articles, string $provider): array
    {
        $articleStates = [];
        foreach ($articles as $article) {
            $slug = $article['slug'] ?? null;
            if ($slug) {
                $articleStates[$slug] = [
                    'status' => 'pending',
                    'attempts' => 0,
                    'last_error' => null,
                    'generated_at' => null,
                    'last_attempt_at' => null,
                ];
            }
        }

        return [
            'batch_id' => $batchId,
            'provider' => $provider,
            'started_at' => now()->toIso8601String(),
            'last_updated' => now()->toIso8601String(),
            'articles' => $articleStates,
        ];
    }

    /**
     * Save batch progress state to the state file.
     */
    protected function saveBatchProgress(string $batchId, array $state): void
    {
        File::put($this->getProgressPath($batchId), json_encode($state, JSON_PRETTY_PRINT));
    }

    /**
     * Get the progress state file path for a batch.
     */
    protected function getProgressPath(string $batchId): string
    {
        return base_path("{$this->batchPath}/{$batchId}.progress.json");
    }

    /**
     * Generate a single article.
     */
    public function generateArticle(
        array $article,
        array $spec,
        string $promptTemplate,
        string $provider = 'gemini'
    ): string {
        $prompt = $this->buildPrompt($article, $spec, $promptTemplate);

        $response = $this->ai->provider($provider)->generate(
            systemPrompt: 'You are a professional content writer for Host Hub.',
            userPrompt: $prompt,
            config: [
                'temperature' => 0.7,
                'max_tokens' => 4000,
            ]
        );

        return $response->content;
    }

    /**
     * Refine a draft using Claude for quality improvement.
     */
    public function refineDraft(string $draftPath): string
    {
        if (! File::exists($draftPath)) {
            throw new \InvalidArgumentException("Draft not found: {$draftPath}");
        }

        $draft = File::get($draftPath);
        $refinementPrompt = $this->loadPromptTemplate('quality-refinement');

        $prompt = str_replace(
            ['{{DRAFT_CONTENT}}'],
            [$draft],
            $refinementPrompt
        );

        $response = $this->ai->claude()->generate($prompt, [
            'temperature' => 0.3,
            'max_tokens' => 4000,
        ]);

        return $response->content;
    }

    /**
     * Validate a draft against quality gates.
     */
    public function validateDraft(string $draftPath): array
    {
        if (! File::exists($draftPath)) {
            return ['valid' => false, 'errors' => ['Draft file not found']];
        }

        $content = File::get($draftPath);
        $errors = [];
        $warnings = [];

        // Word count check
        $wordCount = str_word_count(strip_tags($content));
        if ($wordCount < 600) {
            $errors[] = "Word count too low: {$wordCount} (minimum 600)";
        } elseif ($wordCount > 1500) {
            $warnings[] = "Word count high: {$wordCount} (target 800-1200)";
        }

        // UK English spelling check (basic)
        $usSpellings = ['color', 'customize', 'organize', 'optimize', 'analyze'];
        foreach ($usSpellings as $us) {
            if (stripos($content, $us) !== false) {
                $errors[] = "US spelling detected: '{$us}' - use UK spelling";
            }
        }

        // Check for banned words
        $bannedWords = ['leverage', 'utilize', 'synergy', 'cutting-edge', 'revolutionary', 'seamless', 'robust'];
        foreach ($bannedWords as $banned) {
            if (stripos($content, $banned) !== false) {
                $errors[] = "Banned word detected: '{$banned}'";
            }
        }

        // Check for required sections
        if (stripos($content, '## ') === false && stripos($content, '### ') === false) {
            $errors[] = 'No headings found - article needs structure';
        }

        // Check for FAQ section
        if (stripos($content, 'FAQ') === false && stripos($content, 'frequently asked') === false) {
            $warnings[] = 'No FAQ section found';
        }

        return [
            'valid' => empty($errors),
            'word_count' => $wordCount,
            'errors' => $errors,
            'warnings' => $warnings,
        ];
    }

    /**
     * Parse a batch specification markdown file.
     */
    protected function parseBatchSpec(string $content): array
    {
        $spec = [
            'service' => null,
            'category' => null,
            'priority' => null,
            'variables' => [],
            'articles' => [],
        ];

        // Extract header metadata
        if (preg_match('/\*\*Service:\*\*\s*(.+)/i', $content, $m)) {
            $spec['service'] = trim($m[1]);
        }
        if (preg_match('/\*\*Category:\*\*\s*(.+)/i', $content, $m)) {
            $spec['category'] = trim($m[1]);
        }
        if (preg_match('/\*\*Priority:\*\*\s*(.+)/i', $content, $m)) {
            $spec['priority'] = strtolower(trim(explode('(', $m[1])[0]));
        }

        // Extract generation variables from YAML block
        if (preg_match('/```yaml\s*(SERVICE_NAME:.*?)```/s', $content, $m)) {
            try {
                $spec['variables'] = Yaml::parse($m[1]);
            } catch (\Exception $e) {
                // Ignore parse errors
            }
        }

        // Extract articles (YAML blocks after ### Article headers)
        preg_match_all('/### Article \d+:.*?\n```yaml\s*(.+?)```/s', $content, $matches);
        foreach ($matches[1] as $yaml) {
            try {
                $article = Yaml::parse($yaml);
                if (isset($article['SLUG'])) {
                    $spec['articles'][] = array_change_key_case($article, CASE_LOWER);
                }
            } catch (\Exception $e) {
                // Skip malformed YAML
            }
        }

        return $spec;
    }

    /**
     * Load a prompt template.
     */
    protected function loadPromptTemplate(string $name): string
    {
        $file = base_path("{$this->promptPath}/{$name}.md");

        if (! File::exists($file)) {
            throw new \InvalidArgumentException("Prompt template not found: {$name}");
        }

        return File::get($file);
    }

    /**
     * Build the full prompt for an article.
     */
    protected function buildPrompt(array $article, array $spec, string $template): string
    {
        $vars = array_merge($spec['variables'] ?? [], $article);

        // Replace template variables
        $prompt = $template;
        foreach ($vars as $key => $value) {
            if (is_string($value)) {
                $placeholder = '{{'.strtoupper($key).'}}';
                $prompt = str_replace($placeholder, $value, $prompt);
            } elseif (is_array($value)) {
                $placeholder = '{{'.strtoupper($key).'}}';
                $prompt = str_replace($placeholder, implode(', ', $value), $prompt);
            }
        }

        // Build outline section
        if (isset($article['outline'])) {
            $outlineText = $this->formatOutline($article['outline']);
            $prompt = str_replace('{{OUTLINE}}', $outlineText, $prompt);
        }

        return $prompt;
    }

    /**
     * Format an outline array into readable text.
     */
    protected function formatOutline(array $outline, int $level = 0): string
    {
        $text = '';
        $indent = str_repeat('  ', $level);

        foreach ($outline as $key => $value) {
            if (is_array($value)) {
                $text .= "{$indent}- {$key}:\n";
                $text .= $this->formatOutline($value, $level + 1);
            } else {
                $text .= "{$indent}- {$value}\n";
            }
        }

        return $text;
    }

    /**
     * Get the draft file path for an article.
     */
    protected function getDraftPath(array $spec, string $slug): string
    {
        $service = strtolower($spec['service'] ?? 'general');
        $category = strtolower($spec['category'] ?? 'general');

        // Map service to folder
        $serviceFolder = match ($service) {
            'host link', 'host bio' => 'bio',
            'host social' => 'social',
            'host analytics' => 'analytics',
            'host trust' => 'trust',
            'host notify' => 'notify',
            default => 'general',
        };

        // Map category to subfolder
        $categoryFolder = match (true) {
            str_contains($category, 'getting started') => 'getting-started',
            str_contains($category, 'blog') => 'blog',
            str_contains($category, 'api') => 'api',
            str_contains($category, 'integration') => 'integrations',
            default => str_replace(' ', '-', $category),
        };

        return base_path("{$this->draftsPath}/help/{$categoryFolder}/{$slug}.md");
    }

    /**
     * Save a draft to file.
     */
    protected function saveDraft(string $path, string $content, array $article): void
    {
        $dir = dirname($path);
        if (! File::isDirectory($dir)) {
            File::makeDirectory($dir, 0755, true);
        }

        // Add frontmatter
        $frontmatter = $this->buildFrontmatter($article);
        $fullContent = "---\n{$frontmatter}---\n\n{$content}";

        File::put($path, $fullContent);
    }

    /**
     * Build YAML frontmatter for a draft.
     */
    protected function buildFrontmatter(array $article): string
    {
        $meta = [
            'title' => $article['title'] ?? '',
            'slug' => $article['slug'] ?? '',
            'status' => 'draft',
            'difficulty' => $article['difficulty'] ?? 'beginner',
            'reading_time' => $article['reading_time'] ?? 5,
            'primary_keyword' => $article['primary_keyword'] ?? '',
            'generated_at' => now()->toIso8601String(),
        ];

        return Yaml::dump($meta);
    }
}
