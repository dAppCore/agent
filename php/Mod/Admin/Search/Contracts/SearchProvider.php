<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Search\Contracts;

interface SearchProvider
{
    /**
     * @return array<int, array<string, mixed>|object>
     */
    public function search(string $query): array;

    public function name(): string;

    public function icon(): string;
}
