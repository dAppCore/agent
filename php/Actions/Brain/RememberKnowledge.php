<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Brain;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;

/**
 * Store a memory in the shared OpenBrain knowledge store.
 *
 * Persists content with embeddings to both MariaDB and Qdrant.
 * Handles supersession (replacing old memories) and expiry.
 *
 * Usage:
 *   $memory = RememberKnowledge::run($data, 1, 'virgil');
 */
class RememberKnowledge
{
    use Action;

    public function __construct(
        private BrainService $brain,
    ) {}

    /**
     * @param  array{content: string, type: string, tags?: array, project?: string, confidence?: float, supersedes?: string, expires_in?: int}  $data
     * @return BrainMemory The created memory
     *
     * @throws \InvalidArgumentException
     * @throws \RuntimeException
     */
    public function handle(array $data, int $workspaceId, string $agentId = 'anonymous'): BrainMemory
    {
        $content = $data['content'] ?? null;
        if (! is_string($content) || $content === '') {
            throw new \InvalidArgumentException('content is required and must be a non-empty string');
        }
        if (mb_strlen($content) > 50000) {
            throw new \InvalidArgumentException('content must not exceed 50,000 characters');
        }

        $type = $data['type'] ?? null;
        if (! is_string($type) || ! in_array($type, BrainMemory::VALID_TYPES, true)) {
            throw new \InvalidArgumentException(
                sprintf('type must be one of: %s', implode(', ', BrainMemory::VALID_TYPES))
            );
        }

        $confidence = (float) ($data['confidence'] ?? 0.8);
        if ($confidence < 0.0 || $confidence > 1.0) {
            throw new \InvalidArgumentException('confidence must be between 0.0 and 1.0');
        }

        $tags = $data['tags'] ?? null;
        if (is_array($tags)) {
            foreach ($tags as $tag) {
                if (! is_string($tag)) {
                    throw new \InvalidArgumentException('Each tag must be a string');
                }
            }
        }

        $supersedes = $data['supersedes'] ?? null;
        if ($supersedes !== null) {
            $existing = BrainMemory::where('id', $supersedes)
                ->where('workspace_id', $workspaceId)
                ->first();

            if (! $existing) {
                throw new \InvalidArgumentException("Memory '{$supersedes}' not found in this workspace");
            }
        }

        $expiresIn = isset($data['expires_in']) ? (int) $data['expires_in'] : null;
        if ($expiresIn !== null && $expiresIn < 1) {
            throw new \InvalidArgumentException('expires_in must be at least 1 hour');
        }

        return $this->brain->remember([
            'workspace_id' => $workspaceId,
            'agent_id' => $agentId,
            'type' => $type,
            'content' => $content,
            'tags' => $tags,
            'project' => $data['project'] ?? null,
            'confidence' => $confidence,
            'supersedes_id' => $supersedes,
            'expires_at' => $expiresIn ? now()->addHours($expiresIn) : null,
        ]);
    }
}
