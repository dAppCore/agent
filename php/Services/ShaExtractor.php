<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

class ShaExtractor
{
    private const COMMIT_URL_PATTERN = '#https://forge\.lthn\.sh/(\w+)/(\w+)/commit/([0-9a-f]{7,40})#';

    private const BARE_SHA_PATTERN = '#\b([0-9a-f]{7,40})\b#';

    /**
     * @return array{sha:?string, repo:?string, forge_url:?string}
     */
    public function extract(string $modelOutput): array
    {
        if ($modelOutput === '') {
            return [
                'sha' => null,
                'repo' => null,
                'forge_url' => null,
            ];
        }

        $commitMatch = [];
        $bareMatch = [];
        $hasCommitUrl = preg_match(self::COMMIT_URL_PATTERN, $modelOutput, $commitMatch, PREG_OFFSET_CAPTURE) === 1;
        $hasBareSha = preg_match(self::BARE_SHA_PATTERN, $modelOutput, $bareMatch, PREG_OFFSET_CAPTURE) === 1;

        if (! $hasCommitUrl && ! $hasBareSha) {
            return [
                'sha' => null,
                'repo' => null,
                'forge_url' => null,
            ];
        }

        if ($hasCommitUrl && (! $hasBareSha || $commitMatch[0][1] <= $bareMatch[0][1])) {
            return [
                'sha' => $commitMatch[3][0],
                'repo' => "{$commitMatch[1][0]}/{$commitMatch[2][0]}",
                'forge_url' => $commitMatch[0][0],
            ];
        }

        return [
            'sha' => $bareMatch[1][0],
            'repo' => null,
            'forge_url' => null,
        ];
    }
}
