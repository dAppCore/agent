<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Http\Controllers\Api;

use Core\Mod\Agentic\Jobs\DispatchMantisTicketJob;
use Core\Mod\Agentic\Models\AgentProfile;
use Core\Mod\Agentic\Services\ProfileSelector;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Response;
use Illuminate\Routing\Controller;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Validator;

class MantisWebhookController extends Controller
{
    public function receive(Request $request, ProfileSelector $profileSelector): JsonResponse|Response
    {
        if (! $this->authorised($request)) {
            Log::warning('Mantis webhook authentication failed', [
                'event' => $request->input('event'),
                'issue_id' => $request->input('issue.id'),
            ]);

            return response()->json([
                'message' => 'Unauthorised',
            ], 401);
        }

        /** @var array{event: string, issue: array{id: int|string, summary?: string, severity?: string, priority?: string}} $payload */
        $payload = Validator::make($request->all(), [
            'event' => ['required', 'string'],
            'issue' => ['required', 'array'],
            'issue.id' => ['required', 'integer'],
            'issue.summary' => ['sometimes', 'string'],
            'issue.severity' => ['sometimes', 'string'],
            'issue.priority' => ['sometimes', 'string'],
        ])->validate();

        $issueId = (int) $payload['issue']['id'];

        if ($payload['event'] !== 'issue.opened') {
            Log::info('Mantis webhook ignored event', [
                'event' => $payload['event'],
                'issue_id' => $issueId,
            ]);

            return response()->noContent();
        }

        $profile = $profileSelector->pickFor($payload['issue']);

        if (! $profile instanceof AgentProfile) {
            Log::info('Mantis webhook found no matching profile for issue', [
                'issue_id' => $issueId,
            ]);

            return response()->json([
                'status' => 'accepted',
                'dispatched' => false,
            ]);
        }

        DispatchMantisTicketJob::dispatch($issueId);

        Log::info('Mantis webhook dispatched issue', [
            'issue_id' => $issueId,
            'profile_id' => $profile->getKey(),
        ]);

        return response()->json([
            'status' => 'accepted',
            'dispatched' => true,
        ]);
    }

    private function authorised(Request $request): bool
    {
        $expectedSecret = (string) config('agentic.mantis.webhook_secret');
        $providedSecret = (string) $request->header('X-Mantis-Webhook-Secret');

        if ($expectedSecret === '' || $providedSecret === '') {
            return false;
        }

        return hash_equals($expectedSecret, $providedSecret);
    }
}
