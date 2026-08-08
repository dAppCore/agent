<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Resources\Agent;

use Core\Mod\Agentic\Models\AgentSession;

/**
 * sessions://{sessionId}/context — a session's handoff context as markdown:
 * agent, status, duration, its plan, context summary, handoff notes and
 * artefacts. This is what a resuming agent reads to pick the work back up.
 *
 * @example
 * (new SessionContextResource())->read('sessions://abc123/context');
 */
final class SessionContextResource extends AgentResource
{
    public function uriTemplate(): string
    {
        return 'sessions://{sessionId}/context';
    }

    public function matches(string $uri): bool
    {
        return $this->sessionIdFor($uri) !== null;
    }

    public function read(string $uri): ?string
    {
        $sessionId = $this->sessionIdFor($uri);
        if ($sessionId === null) {
            return null;
        }

        $session = AgentSession::where('session_id', $sessionId)->first();
        if (! $session) {
            return null;
        }

        $context = $session->getHandoffContext();

        $markdown = "# Session: {$session->session_id}\n\n";
        $markdown .= "**Agent:** {$session->agent_type}\n";
        $markdown .= "**Status:** {$session->status}\n";
        $markdown .= "**Duration:** {$session->getDurationFormatted()}\n\n";

        if ($session->plan) {
            $markdown .= "## Plan\n\n";
            $markdown .= "**{$session->plan->title}** ({$session->plan->slug})\n\n";
        }

        if (! empty($context['context_summary'])) {
            $markdown .= "## Context Summary\n\n";
            $markdown .= json_encode($context['context_summary'], JSON_PRETTY_PRINT)."\n\n";
        }

        if (! empty($context['handoff_notes'])) {
            $markdown .= "## Handoff Notes\n\n";
            $markdown .= json_encode($context['handoff_notes'], JSON_PRETTY_PRINT)."\n\n";
        }

        if (! empty($context['artifacts'])) {
            $markdown .= "## Artifacts\n\n";
            foreach ($context['artifacts'] as $artifact) {
                $markdown .= "- {$artifact['action']}: {$artifact['path']}\n";
            }
            $markdown .= "\n";
        }

        return $markdown;
    }

    public function entries(): array
    {
        return [];
    }

    private function sessionIdFor(string $uri): ?string
    {
        if (! str_starts_with($uri, 'sessions://')) {
            return null;
        }

        $parts = explode('/', substr($uri, strlen('sessions://')));

        if (count($parts) !== 2 || $parts[1] !== 'context' || $parts[0] === '') {
            return null;
        }

        return $parts[0];
    }
}
