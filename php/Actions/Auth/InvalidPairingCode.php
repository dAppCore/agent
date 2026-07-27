<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Auth;

use RuntimeException;

/**
 * Raised when a device-pairing code is unknown, expired, or already spent.
 */
class InvalidPairingCode extends RuntimeException
{
    public function __construct(string $message = 'Invalid or expired pairing code')
    {
        parent::__construct($message);
    }
}
