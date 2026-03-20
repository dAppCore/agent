<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Mod\Uptelligence\Models\UptelligenceWebhookDelivery;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Routing\Controller;

/**
 * CheckinController — agents call this to discover which repos changed.
 *
 * GET /v1/agent/checkin?agent=cladius&since=1773730000
 *
 * Returns repos that received push events since the given timestamp.
 * Agents use this to pull only what changed, avoiding 50+ git fetch calls.
 */
class CheckinController extends Controller
{
    /**
     * Agent checkin — return repos with push events since timestamp.
     */
    public function checkin(Request $request): JsonResponse
    {
        $since = (int) $request->query('since', '0');
        $agent = $request->query('agent', 'unknown');

        $sinceDate = $since > 0
            ? \Carbon\Carbon::createFromTimestamp($since)
            : now()->subMinutes(5);

        // Query webhook deliveries for push events since the given time.
        // Forgejo sends GitHub-compatible webhooks, so event_type is "github.push.*".
        $deliveries = UptelligenceWebhookDelivery::query()
            ->where('created_at', '>', $sinceDate)
            ->where('event_type', 'like', '%push%')
            ->where('status', '!=', 'failed')
            ->orderBy('created_at', 'asc')
            ->get();

        $changed = [];
        $seen = [];

        foreach ($deliveries as $delivery) {
            $payload = $delivery->payload;
            if (! is_array($payload)) {
                continue;
            }

            // Extract repo name and branch from Forgejo/GitHub push payload
            $repoName = $payload['repository']['name'] ?? null;
            $ref = $payload['ref'] ?? '';
            $sha = $payload['after'] ?? '';

            // Only track pushes to main/master
            if (! $repoName || ! str_ends_with($ref, '/main') && ! str_ends_with($ref, '/master')) {
                continue;
            }

            $branch = basename($ref);

            // Deduplicate — only latest push per repo
            if (isset($seen[$repoName])) {
                continue;
            }
            $seen[$repoName] = true;

            $changed[] = [
                'repo' => $repoName,
                'branch' => $branch,
                'sha' => $sha,
            ];
        }

        return response()->json([
            'changed' => $changed,
            'timestamp' => now()->timestamp,
            'agent' => $agent,
        ]);
    }
}
