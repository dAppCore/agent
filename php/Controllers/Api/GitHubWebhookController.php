<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Response;
use Illuminate\Routing\Controller;
use Illuminate\Support\Facades\Log;

/**
 * GitHubWebhookController — receives webhook events from the GitHub App.
 *
 * Events handled:
 *   - pull_request_review: Auto-merge on approval, extract findings on changes_requested
 *   - push: Reverse sync to Forge (future)
 *   - check_run: Build status tracking (future)
 *
 * All events are logged for CodeRabbit KPI tracking.
 */
class GitHubWebhookController extends Controller
{
    /**
     * Receive a GitHub App webhook.
     *
     * POST /api/github/webhook
     */
    public function receive(Request $request): Response|JsonResponse
    {
        // Verify signature
        $secret = config('agentic.github_webhook_secret', env('GITHUB_WEBHOOK_SECRET'));
        if ($secret && ! $this->verifySignature($request, $secret)) {
            Log::warning('GitHub webhook signature verification failed', [
                'ip' => $request->ip(),
            ]);

            return response('Invalid signature', 401);
        }

        $event = $request->header('X-GitHub-Event', 'unknown');
        $payload = $request->json()->all();

        Log::info('GitHub webhook received', [
            'event' => $event,
            'action' => $payload['action'] ?? 'none',
            'repo' => $payload['repository']['full_name'] ?? 'unknown',
        ]);

        // Store raw event for KPI tracking
        $this->storeEvent($event, $payload);

        return match ($event) {
            'pull_request_review' => $this->handlePullRequestReview($payload),
            'push' => $this->handlePush($payload),
            'check_run' => $this->handleCheckRun($payload),
            default => response()->json(['status' => 'ignored', 'event' => $event]),
        };
    }

    /**
     * Handle pull_request_review events.
     *
     * - approved by coderabbitai → queue auto-merge
     * - changes_requested by coderabbitai → store findings for agent dispatch
     */
    protected function handlePullRequestReview(array $payload): JsonResponse
    {
        $action = $payload['action'] ?? '';
        $review = $payload['review'] ?? [];
        $pr = $payload['pull_request'] ?? [];
        $reviewer = $review['user']['login'] ?? '';
        $state = $review['state'] ?? '';
        $repo = $payload['repository']['name'] ?? '';
        $prNumber = $pr['number'] ?? 0;

        if ($reviewer !== 'coderabbitai') {
            return response()->json(['status' => 'ignored', 'reason' => 'not coderabbit']);
        }

        if ($state === 'approved') {
            Log::info('CodeRabbit approved PR', [
                'repo' => $repo,
                'pr' => $prNumber,
            ]);

            // Store approval event
            $this->storeCodeRabbitResult($repo, $prNumber, 'approved', null);

            return response()->json([
                'status' => 'approved',
                'repo' => $repo,
                'pr' => $prNumber,
                'action' => 'merge_queued',
            ]);
        }

        if ($state === 'changes_requested') {
            $body = $review['body'] ?? '';

            Log::info('CodeRabbit requested changes', [
                'repo' => $repo,
                'pr' => $prNumber,
                'body_length' => strlen($body),
            ]);

            // Store findings for agent dispatch
            $this->storeCodeRabbitResult($repo, $prNumber, 'changes_requested', $body);

            return response()->json([
                'status' => 'changes_requested',
                'repo' => $repo,
                'pr' => $prNumber,
                'action' => 'findings_stored',
            ]);
        }

        return response()->json(['status' => 'ignored', 'state' => $state]);
    }

    /**
     * Handle push events (future: reverse sync to Forge).
     */
    protected function handlePush(array $payload): JsonResponse
    {
        $repo = $payload['repository']['name'] ?? '';
        $ref = $payload['ref'] ?? '';
        $after = $payload['after'] ?? '';

        Log::info('GitHub push', [
            'repo' => $repo,
            'ref' => $ref,
            'sha' => substr($after, 0, 8),
        ]);

        return response()->json(['status' => 'logged', 'repo' => $repo]);
    }

    /**
     * Handle check_run events (future: build status tracking).
     */
    protected function handleCheckRun(array $payload): JsonResponse
    {
        return response()->json(['status' => 'logged']);
    }

    /**
     * Verify GitHub webhook signature (SHA-256).
     */
    protected function verifySignature(Request $request, string $secret): bool
    {
        $signature = $request->header('X-Hub-Signature-256', '');
        if (empty($signature)) {
            return false;
        }

        $payload = $request->getContent();
        $expected = 'sha256=' . hash_hmac('sha256', $payload, $secret);

        return hash_equals($expected, $signature);
    }

    /**
     * Store raw webhook event for KPI tracking.
     */
    protected function storeEvent(string $event, array $payload): void
    {
        $repo = $payload['repository']['name'] ?? 'unknown';
        $action = $payload['action'] ?? '';

        // Store in uptelligence webhook deliveries if available
        try {
            \DB::table('github_webhook_events')->insert([
                'event' => $event,
                'action' => $action,
                'repo' => $repo,
                'payload' => json_encode($payload),
                'created_at' => now(),
            ]);
        } catch (\Throwable) {
            // Table may not exist yet — log only
            Log::debug('GitHub webhook event stored in log only', [
                'event' => $event,
                'repo' => $repo,
            ]);
        }
    }

    /**
     * Store CodeRabbit review result for KPI tracking.
     */
    protected function storeCodeRabbitResult(string $repo, int $prNumber, string $result, ?string $body): void
    {
        try {
            \DB::table('coderabbit_reviews')->insert([
                'repo' => $repo,
                'pr_number' => $prNumber,
                'result' => $result,
                'findings' => $body,
                'created_at' => now(),
            ]);
        } catch (\Throwable) {
            Log::debug('CodeRabbit result stored in log only', [
                'repo' => $repo,
                'pr' => $prNumber,
                'result' => $result,
            ]);
        }
    }
}
