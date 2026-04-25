<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Pipeline;

use Core\Mod\Agentic\Services\ForgejoService;
use Illuminate\Support\Facades\Http;
use ReflectionClass;
use RuntimeException;

final class ForgejoMetaReader implements MetaReader
{
    public function __construct(
        private ForgejoService $forgejo,
        private ?string $owner = null,
        private ?string $repo = null,
    ) {}

    public function getPRMeta(int $prNumber): PRMeta
    {
        [$owner, $repo] = $this->resolveRepo();

        $pr = $this->forgejo->getPullRequest($owner, $repo, $prNumber);
        $headSha = $this->stringOrNull($pr['head']['sha'] ?? null);
        $status = $headSha === null ? [] : $this->forgejo->getCombinedStatus($owner, $repo, $headSha);

        $reviewThreadsTotal = $this->extractReviewThreadsTotal($pr);

        return new PRMeta(
            state: $this->extractPRState($pr),
            mergeability: $this->extractMergeability($pr),
            headSha: $headSha,
            headDate: $this->extractHeadDate($pr, $status),
            baseBranch: $this->stringOrNull($pr['base']['ref'] ?? null),
            headBranch: $this->stringOrNull($pr['head']['ref'] ?? null),
            checkStatuses: $this->extractCheckStatuses($status),
            reviewThreadsTotal: $reviewThreadsTotal,
            reviewThreadsResolved: $this->extractResolvedThreadCount($pr, $reviewThreadsTotal),
            hasEyesReaction: $this->hasEyesReaction($pr),
        );
    }

    public function getEpicMeta(int $issueNumber): EpicMeta
    {
        [$owner, $repo] = $this->resolveRepo();

        $epic = $this->forgejo->getIssue($owner, $repo, $issueNumber);

        return new EpicMeta(
            state: $this->extractIssueLifecycle($epic),
            children: $this->extractEpicChildren($owner, $repo, $epic),
        );
    }

    public function getIssueState(int $issueNumber): IssueState
    {
        [$owner, $repo] = $this->resolveRepo();

        $issue = $this->forgejo->getIssue($owner, $repo, $issueNumber);

        return new IssueState(
            state: $this->extractIssueLifecycle($issue),
            title: (string) ($issue['title'] ?? ''),
            labels: $this->extractLabels($issue),
            assignee: $this->extractAssignee($issue),
        );
    }

    public function getCommentReactions(int $issueNumber, int $commentNumber): Reactions
    {
        [$owner, $repo] = $this->resolveRepo();

        // The reactions API is comment-scoped; the issue number remains on the
        // contract for pipeline symmetry and future validation.
        unset($issueNumber);

        // ForgejoService does not currently expose reactions, so reuse its
        // configured base URL and token here and return counts only.
        $reactions = $this->directGet("/repos/{$owner}/{$repo}/issues/comments/{$commentNumber}/reactions");

        return $this->aggregateReactions($reactions);
    }

    /**
     * @return array{0: string, 1: string}
     */
    private function resolveRepo(): array
    {
        if ($this->owner !== null || $this->repo !== null) {
            if ($this->owner === null || $this->repo === null) {
                throw new RuntimeException('ForgejoMetaReader requires both owner and repo when one is provided.');
            }

            return [$this->owner, $this->repo];
        }

        $configuredRepos = array_values(array_filter((array) config('agentic.scan_repos', [])));

        if (count($configuredRepos) !== 1) {
            throw new RuntimeException('ForgejoMetaReader requires an explicit owner/repo or exactly one configured agentic.scan_repos entry.');
        }

        $repoSpec = trim((string) $configuredRepos[0]);
        $parts = explode('/', $repoSpec, 2);

        if (count($parts) !== 2 || $parts[0] === '' || $parts[1] === '') {
            throw new RuntimeException("Invalid Forgejo repository spec: {$repoSpec}");
        }

        return [$parts[0], $parts[1]];
    }

    /**
     * @param  array<string, mixed>  $pr
     */
    private function extractPRState(array $pr): string
    {
        if (($pr['merged'] ?? false) === true) {
            return 'merged';
        }

        $state = strtolower((string) ($pr['state'] ?? ''));

        return $state === '' ? 'unknown' : $state;
    }

    /**
     * @param  array<string, mixed>  $pr
     */
    private function extractMergeability(array $pr): string
    {
        if (($pr['mergeable'] ?? null) === true) {
            return 'mergeable';
        }

        if (($pr['mergeable'] ?? null) === false) {
            return 'conflicting';
        }

        return match (strtolower((string) ($pr['mergeable_state'] ?? ''))) {
            'clean', 'mergeable' => 'mergeable',
            'dirty', 'conflicting' => 'conflicting',
            default => 'unknown',
        };
    }

    /**
     * @param  array<string, mixed>  $pr
     * @param  array<string, mixed>  $status
     */
    private function extractHeadDate(array $pr, array $status): ?string
    {
        $firstStatus = $status['statuses'][0] ?? [];

        return $this->stringOrNull(
            $pr['head']['date']
            ?? $pr['head']['updated_at']
            ?? $pr['head']['repo']['updated_at']
            ?? $pr['head']['repo']['pushed_at']
            ?? $firstStatus['updated_at']
            ?? $firstStatus['created_at']
            ?? $pr['updated_at']
            ?? null,
        );
    }

    /**
     * @param  array<string, mixed>  $status
     * @return array<int, array{name: string, conclusion: string|null, status: string|null}>
     */
    private function extractCheckStatuses(array $status): array
    {
        $statuses = $status['statuses'] ?? [];

        if (! is_array($statuses)) {
            return [];
        }

        $checks = [];

        foreach ($statuses as $entry) {
            if (! is_array($entry)) {
                continue;
            }

            $rawState = strtolower((string) ($entry['status'] ?? $entry['state'] ?? $entry['conclusion'] ?? ''));
            $name = (string) ($entry['context'] ?? $entry['name'] ?? '');

            $checks[] = [
                'name' => $name,
                'conclusion' => $this->mapCheckConclusion($rawState),
                'status' => $this->mapCheckStatus($rawState),
            ];
        }

        return $checks;
    }

    private function mapCheckConclusion(string $rawState): ?string
    {
        return match ($rawState) {
            'success', 'failure', 'error' => $rawState === 'error' ? 'failure' : $rawState,
            default => null,
        };
    }

    private function mapCheckStatus(string $rawState): ?string
    {
        return match ($rawState) {
            'success', 'failure', 'error' => 'completed',
            'pending', 'queued' => 'queued',
            'running', 'in_progress' => 'in_progress',
            default => $rawState === '' ? null : $rawState,
        };
    }

    /**
     * @param  array<string, mixed>  $pr
     */
    private function extractReviewThreadsTotal(array $pr): int
    {
        foreach ([
            $pr['review_threads_total'] ?? null,
            $pr['review_comments'] ?? null,
            $pr['comments'] ?? null,
        ] as $candidate) {
            $value = $this->intOrNull($candidate);

            if ($value !== null) {
                return $value;
            }
        }

        return 0;
    }

    /**
     * @param  array<string, mixed>  $pr
     */
    private function extractResolvedThreadCount(array $pr, int $reviewThreadsTotal): int
    {
        $resolved = $this->intOrNull($pr['review_threads_resolved'] ?? $pr['resolved_review_comments'] ?? null);

        if ($resolved !== null) {
            return $resolved;
        }

        $unresolved = $this->intOrNull($pr['review_threads_unresolved'] ?? $pr['unresolved_review_comments'] ?? null);

        if ($unresolved !== null) {
            return max(0, $reviewThreadsTotal - $unresolved);
        }

        return 0;
    }

    /**
     * @param  array<string, mixed>  $pr
     */
    private function hasEyesReaction(array $pr): bool
    {
        return ($this->intOrNull($pr['reactions']['eyes'] ?? null) ?? 0) > 0;
    }

    /**
     * @param  array<string, mixed>  $issue
     */
    private function extractIssueLifecycle(array $issue): string
    {
        $state = strtolower((string) ($issue['state'] ?? ''));

        return $state === '' ? 'unknown' : $state;
    }

    /**
     * @param  array<string, mixed>  $issue
     * @return array<int, string>
     */
    private function extractLabels(array $issue): array
    {
        $labels = $issue['labels'] ?? [];

        if (! is_array($labels)) {
            return [];
        }

        $names = [];

        foreach ($labels as $label) {
            if (! is_array($label)) {
                continue;
            }

            $name = trim((string) ($label['name'] ?? ''));

            if ($name !== '') {
                $names[] = $name;
            }
        }

        return $names;
    }

    /**
     * @param  array<string, mixed>  $issue
     */
    private function extractAssignee(array $issue): ?string
    {
        return $this->stringOrNull(
            $issue['assignee']['login']
            ?? $issue['assignees'][0]['login']
            ?? null,
        );
    }

    /**
     * @param  array<string, mixed>  $epic
     * @return array<int, EpicChild>
     */
    private function extractEpicChildren(string $owner, string $repo, array $epic): array
    {
        $rawChildren = $epic['subtasks'] ?? $epic['sub_issues'] ?? null;

        if (! is_array($rawChildren)) {
            // Native Forgejo issue payloads do not consistently expose
            // tasklist-style children structurally, so body parsing remains out
            // of scope here by design.
            return [];
        }

        $needsStateLookup = false;

        foreach ($rawChildren as $rawChild) {
            if (! is_array($rawChild)) {
                continue;
            }

            if (! isset($rawChild['state'])) {
                $needsStateLookup = true;
                break;
            }
        }

        $issueLookup = $needsStateLookup ? $this->buildIssueLookup($owner, $repo) : [];
        $children = [];

        foreach ($rawChildren as $rawChild) {
            if (! is_array($rawChild)) {
                continue;
            }

            $issueId = $this->extractIssueId($rawChild);

            if ($issueId === null) {
                continue;
            }

            $lookup = $issueLookup[$issueId] ?? [];

            $children[] = new EpicChild(
                issueId: $issueId,
                state: $this->extractChildState($rawChild, $lookup),
                checkedBool: $this->extractCheckedFlag($rawChild),
                linkedPrNumberOrNull: $this->extractLinkedPRNumber($rawChild, $lookup),
            );
        }

        return $children;
    }

    /**
     * @return array<int, array<string, mixed>>
     */
    private function buildIssueLookup(string $owner, string $repo): array
    {
        $lookup = [];

        foreach ($this->forgejo->listIssues($owner, $repo, 'all') as $issue) {
            if (! is_array($issue)) {
                continue;
            }

            $issueId = $this->intOrNull($issue['number'] ?? null);

            if ($issueId === null) {
                continue;
            }

            $lookup[$issueId] = $issue;
        }

        return $lookup;
    }

    /**
     * @param  array<string, mixed>  $child
     */
    private function extractIssueId(array $child): ?int
    {
        return $this->intOrNull(
            $child['issue_id']
            ?? $child['number']
            ?? $child['issue']['number']
            ?? $child['id']
            ?? null,
        );
    }

    /**
     * @param  array<string, mixed>  $child
     * @param  array<string, mixed>  $lookup
     */
    private function extractChildState(array $child, array $lookup): string
    {
        $state = strtolower((string) ($child['state'] ?? $lookup['state'] ?? ''));

        return $state === '' ? 'unknown' : $state;
    }

    /**
     * @param  array<string, mixed>  $child
     */
    private function extractCheckedFlag(array $child): bool
    {
        foreach ([
            $child['checked'] ?? null,
            $child['checked_bool'] ?? null,
            $child['is_checked'] ?? null,
            $child['completed'] ?? null,
            $child['done'] ?? null,
        ] as $candidate) {
            $bool = $this->boolOrNull($candidate);

            if ($bool !== null) {
                return $bool;
            }
        }

        return false;
    }

    /**
     * @param  array<string, mixed>  $child
     * @param  array<string, mixed>  $lookup
     */
    private function extractLinkedPRNumber(array $child, array $lookup): ?int
    {
        foreach ([
            $child['linked_pr_number_or_null'] ?? null,
            $child['linked_pr_number'] ?? null,
            $child['linked_pull_request_number'] ?? null,
            $child['pull_request']['number'] ?? null,
            $child['linked_pull_request']['number'] ?? null,
            $lookup['pull_request']['number'] ?? null,
            $lookup['linked_pull_request']['number'] ?? null,
        ] as $candidate) {
            $value = $this->intOrNull($candidate);

            if ($value !== null) {
                return $value;
            }
        }

        return null;
    }

    /**
     * @param  array<int, mixed>  $reactions
     */
    private function aggregateReactions(array $reactions): Reactions
    {
        $counts = [
            '+1' => 0,
            '-1' => 0,
            'laugh' => 0,
            'hooray' => 0,
            'confused' => 0,
            'heart' => 0,
            'rocket' => 0,
            'eyes' => 0,
        ];

        foreach ($reactions as $reaction) {
            if (! is_array($reaction)) {
                continue;
            }

            $content = strtolower((string) ($reaction['content'] ?? ''));

            if (array_key_exists($content, $counts)) {
                $counts[$content]++;
            }
        }

        return new Reactions(
            plusOne: $counts['+1'],
            minusOne: $counts['-1'],
            laugh: $counts['laugh'],
            hooray: $counts['hooray'],
            confused: $counts['confused'],
            heart: $counts['heart'],
            rocket: $counts['rocket'],
            eyes: $counts['eyes'],
        );
    }

    /**
     * @return array<int, mixed>
     */
    private function directGet(string $path): array
    {
        $response = Http::withToken($this->forgejoToken())
            ->acceptJson()
            ->timeout(15)
            ->get($this->forgejoBaseUrl().'/api/v1'.$path);

        if (! $response->successful()) {
            throw new RuntimeException("Forgejo API GET {$path} failed: {$response->status()}");
        }

        $json = $response->json();

        return is_array($json) ? $json : [];
    }

    private function forgejoBaseUrl(): string
    {
        return rtrim((string) $this->readForgejoProperty('baseUrl'), '/');
    }

    private function forgejoToken(): string
    {
        return (string) $this->readForgejoProperty('token');
    }

    private function readForgejoProperty(string $property): mixed
    {
        $reflection = new ReflectionClass($this->forgejo);

        do {
            if ($reflection->hasProperty($property)) {
                $prop = $reflection->getProperty($property);
                $prop->setAccessible(true);

                return $prop->getValue($this->forgejo);
            }
        } while ($reflection = $reflection->getParentClass());

        throw new RuntimeException("Unable to read ForgejoService::\${$property}");
    }

    private function intOrNull(mixed $value): ?int
    {
        if (is_int($value)) {
            return $value;
        }

        if (is_numeric($value)) {
            return (int) $value;
        }

        return null;
    }

    private function boolOrNull(mixed $value): ?bool
    {
        if (is_bool($value)) {
            return $value;
        }

        if (is_string($value)) {
            return match (strtolower($value)) {
                '1', 'true', 'yes', 'x' => true,
                '0', 'false', 'no', '' => false,
                default => null,
            };
        }

        if (is_int($value)) {
            return $value !== 0;
        }

        return null;
    }

    private function stringOrNull(mixed $value): ?string
    {
        if ($value === null) {
            return null;
        }

        $string = trim((string) $value);

        return $string === '' ? null : $string;
    }
}
