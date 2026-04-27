<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Front\Mcp\Contracts;

use Core\Front\Mcp\McpContext;

interface McpToolHandler
{
    public static function schema(): array;

    public function handle(array $args, McpContext $context): array;
}
