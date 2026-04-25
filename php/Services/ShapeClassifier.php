<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use ArrayAccess;

class ShapeClassifier
{
    public const CLASS_A = 'A';

    public const CLASS_B = 'B';

    public const CLASS_C = 'C';

    /**
     * @var list<string>
     */
    private const ESCALATION_CAPABILITIES = [
        'security',
        'crypto',
        'core',
    ];

    /**
     * @param  array<string, mixed>|object  $ticket
     * @return 'A'|'B'|'C'
     */
    public function classify(mixed $ticket): string
    {
        $severity = $this->stringValue($ticket, 'severity');
        $priority = $this->stringValue($ticket, 'priority');

        if ($severity === 'critical' || $priority === 'urgent') {
            return self::CLASS_A;
        }

        if (array_intersect($this->normaliseTags($this->value($ticket, 'tags')), self::ESCALATION_CAPABILITIES) !== []) {
            return self::CLASS_A;
        }

        if ($severity === 'major' || $priority === 'high') {
            return self::CLASS_B;
        }

        return self::CLASS_C;
    }

    /**
     * @param  array<string, mixed>|object  $ticket
     */
    private function stringValue(mixed $ticket, string $key): string
    {
        $value = $this->value($ticket, $key);

        return is_string($value)
            ? strtolower(trim($value))
            : '';
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
}
