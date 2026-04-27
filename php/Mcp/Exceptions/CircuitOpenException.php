<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use RuntimeException;

final class CircuitOpenException extends RuntimeException
{
    public function __construct(
        public readonly string $service,
        string $message = '',
    ) {
        parent::__construct($message !== '' ? $message : sprintf(
            "Service '%s' is temporarily unavailable. Please try again later.",
            $service,
        ));
    }
}
