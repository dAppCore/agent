<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Illuminate\Support\Facades\Http;

/**
 * Forgejo REST API client for agent orchestration.
 *
 * Wraps the Forgejo v1 API for issue management, pull requests,
 * commit statuses, and branch operations.
 */
class ForgejoService
{
    public function __construct(
        private string $baseUrl,
        private string $token,
    ) {}

    /**
     * List issues for a repository.
     *
     * @return array<int, array<string, mixed>>
     */
    public function listIssues(string $owner, string $repo, string $state = 'open', ?string $label = null): array
    {
        $query = ['state' => $state, 'type' => 'issues'];

        if ($label !== null) {
            $query['labels'] = $label;
        }

        return $this->get("/repos/{$owner}/{$repo}/issues", $query);
    }

    /**
     * Get a single issue by number.
     *
     * @return array<string, mixed>
     */
    public function getIssue(string $owner, string $repo, int $number): array
    {
        return $this->get("/repos/{$owner}/{$repo}/issues/{$number}");
    }

    /**
     * Create a comment on an issue.
     *
     * @return array<string, mixed>
     */
    public function createComment(string $owner, string $repo, int $issueNumber, string $body): array
    {
        return $this->post("/repos/{$owner}/{$repo}/issues/{$issueNumber}/comments", [
            'body' => $body,
        ]);
    }

    /**
     * Add labels to an issue.
     *
     * @param  array<int>  $labelIds
     * @return array<int, array<string, mixed>>
     */
    public function addLabels(string $owner, string $repo, int $issueNumber, array $labelIds): array
    {
        return $this->post("/repos/{$owner}/{$repo}/issues/{$issueNumber}/labels", [
            'labels' => $labelIds,
        ]);
    }

    /**
     * List pull requests for a repository.
     *
     * @return array<int, array<string, mixed>>
     */
    public function listPullRequests(string $owner, string $repo, string $state = 'all'): array
    {
        return $this->get("/repos/{$owner}/{$repo}/pulls", ['state' => $state]);
    }

    /**
     * Get a single pull request by number.
     *
     * @return array<string, mixed>
     */
    public function getPullRequest(string $owner, string $repo, int $number): array
    {
        return $this->get("/repos/{$owner}/{$repo}/pulls/{$number}");
    }

    /**
     * Get the combined commit status for a ref.
     *
     * @return array<string, mixed>
     */
    public function getCombinedStatus(string $owner, string $repo, string $sha): array
    {
        return $this->get("/repos/{$owner}/{$repo}/commits/{$sha}/status");
    }

    /**
     * Merge a pull request.
     *
     * @param  string  $method  One of: merge, rebase, rebase-merge, squash, fast-forward-only
     *
     * @throws \RuntimeException
     */
    public function mergePullRequest(string $owner, string $repo, int $number, string $method = 'merge'): void
    {
        $response = $this->request()
            ->post($this->url("/repos/{$owner}/{$repo}/pulls/{$number}/merge"), [
                'Do' => $method,
            ]);

        if (! $response->successful()) {
            throw new \RuntimeException(
                "Failed to merge PR #{$number}: {$response->status()} {$response->body()}"
            );
        }
    }

    /**
     * Create a branch in a repository.
     *
     * @return array<string, mixed>
     */
    public function createBranch(string $owner, string $repo, string $name, string $from = 'main'): array
    {
        return $this->post("/repos/{$owner}/{$repo}/branches", [
            'new_branch_name' => $name,
            'old_branch_name' => $from,
        ]);
    }

    /**
     * Build an authenticated HTTP client.
     */
    private function request(): \Illuminate\Http\Client\PendingRequest
    {
        return Http::withToken($this->token)
            ->acceptJson()
            ->timeout(15);
    }

    /**
     * Build the full API URL for a path.
     */
    private function url(string $path): string
    {
        return "{$this->baseUrl}/api/v1{$path}";
    }

    /**
     * Perform a GET request and return decoded JSON.
     *
     * @param  array<string, mixed>  $query
     * @return array<string, mixed>
     *
     * @throws \RuntimeException
     */
    private function get(string $path, array $query = []): array
    {
        $response = $this->request()->get($this->url($path), $query);

        if (! $response->successful()) {
            throw new \RuntimeException(
                "Forgejo API GET {$path} failed: {$response->status()}"
            );
        }

        return $response->json();
    }

    /**
     * Perform a POST request and return decoded JSON.
     *
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     *
     * @throws \RuntimeException
     */
    private function post(string $path, array $data = []): array
    {
        $response = $this->request()->post($this->url($path), $data);

        if (! $response->successful()) {
            throw new \RuntimeException(
                "Forgejo API POST {$path} failed: {$response->status()}"
            );
        }

        return $response->json();
    }
}
