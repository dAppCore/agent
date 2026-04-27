<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use ArrayAccess;
use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;

class ProfileSelector
{
    private ShapeClassifier $shapeClassifier;

    public function __construct(?ShapeClassifier $shapeClassifier = null)
    {
        $this->shapeClassifier = $shapeClassifier ?? new ShapeClassifier;
    }

    /**
     * @param  array<string, mixed>|object  $ticket
     */
    public function pickFor(mixed $ticket): ?AgentProfile
    {
        return match ($this->shapeClassifier->classify($ticket)) {
            ShapeClassifier::CLASS_A => $this->pickClassA($ticket),
            ShapeClassifier::CLASS_B => $this->pickClassB($ticket),
            default => $this->pickClassC(),
        };
    }

    /**
     * @param  array<string, mixed>|object  $ticket
     */
    private function pickClassA(mixed $ticket): ?AgentProfile
    {
        $requiredCapabilities = $this->requiredCapabilities($ticket);
        if ($requiredCapabilities === []) {
            return null;
        }

        $profiles = AgentProfile::query()
            ->active()
            ->orderBy('cost_class')
            ->orderBy('id')
            ->get();

        foreach ($profiles as $profile) {
            if ($this->matchesRequiredCapability($profile, $requiredCapabilities)) {
                $this->recordDispatch($profile);

                return $profile->fresh();
            }
        }

        return null;
    }

    /**
     * @param  array<string, mixed>|object  $ticket
     */
    private function pickClassB(mixed $ticket): ?AgentProfile
    {
        $requiredCapabilities = $this->requiredCapabilities($ticket);
        if ($requiredCapabilities === []) {
            return null;
        }

        $profiles = AgentProfile::query()
            ->active()
            ->where('quota_headroom_pct', '>=', 25)
            ->orderByRaw('CASE WHEN last_dispatched_at IS NULL THEN 0 ELSE 1 END')
            ->orderBy('last_dispatched_at')
            ->orderBy('id')
            ->get();

        foreach ($profiles as $profile) {
            if ($this->matchesRequiredCapability($profile, $requiredCapabilities)) {
                $this->recordDispatch($profile);

                return $profile->fresh();
            }
        }

        return null;
    }

    private function pickClassC(): ?AgentProfile
    {
        return DB::transaction(function (): ?AgentProfile {
            $profiles = AgentProfile::query()
                ->active()
                ->orderByRaw('CASE WHEN last_dispatched_at IS NULL THEN 0 ELSE 1 END')
                ->orderBy('last_dispatched_at')
                ->orderBy('id')
                ->lockForUpdate()
                ->get();

            $profile = $profiles->first();
            if (! $profile instanceof AgentProfile) {
                return null;
            }

            $this->recordDispatch($profile, $profiles);

            return $profile->fresh();
        });
    }

    /**
     * @param  list<string>  $requiredCapabilities
     */
    private function matchesRequiredCapability(AgentProfile $profile, array $requiredCapabilities): bool
    {
        $profileCapabilities = $this->normaliseTags($profile->capability_tags);

        return array_intersect($requiredCapabilities, $profileCapabilities) !== [];
    }

    /**
     * @param  array<string, mixed>|object  $ticket
     * @return list<string>
     */
    private function requiredCapabilities(mixed $ticket): array
    {
        return $this->normaliseTags($this->value($ticket, 'tags'));
    }

    /**
     * @return list<string>
     */
    private function normaliseTags(mixed $tags): array
    {
        if (! is_array($tags)) {
            return [];
        }

        $normalised = [];

        foreach ($tags as $key => $value) {
            if (is_string($key) && $value) {
                $tag = strtolower(trim($key));
                if ($tag !== '') {
                    $normalised[] = $tag;
                }
            }

            if (is_string($value)) {
                $tag = strtolower(trim($value));
                if ($tag !== '') {
                    $normalised[] = $tag;
                }
            }
        }

        return array_values(array_unique($normalised));
    }

    /**
     * @param  array<string, mixed>|object  $ticket
     */
    private function value(mixed $ticket, string $key): mixed
    {
        if (is_array($ticket)) {
            return $ticket[$key] ?? null;
        }

        if ($ticket instanceof ArrayAccess && $ticket->offsetExists($key)) {
            return $ticket[$key];
        }

        if (is_object($ticket) && property_exists($ticket, $key)) {
            return $ticket->{$key};
        }

        return null;
    }

    private function recordDispatch(AgentProfile $profile, ?Collection $activeProfiles = null): void
    {
        $profile->forceFill([
            'last_dispatched_at' => $this->dispatchTimestamp($activeProfiles),
        ])->save();
    }

    private function dispatchTimestamp(?Collection $activeProfiles = null): Carbon
    {
        $dispatchAt = now();

        $latestDispatch = $activeProfiles
            ? $activeProfiles
                ->pluck('last_dispatched_at')
                ->filter(static fn (mixed $value): bool => $value instanceof Carbon)
                ->sortBy(static fn (Carbon $value): int => $value->getTimestamp())
                ->last()
            : null;

        if ($latestDispatch instanceof Carbon && $latestDispatch->greaterThanOrEqualTo($dispatchAt)) {
            return $latestDispatch->copy()->addSecond();
        }

        return $dispatchAt;
    }
}
