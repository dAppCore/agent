<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Front\Mcp;

use Closure;

final class McpContext
{
    public function __construct(
        private ?string $sessionId = null,
        private ?object $currentPlan = null,
        private ?Closure $notificationCallback = null,
        private ?Closure $logCallback = null,
    ) {}

    public function getSessionId(): ?string
    {
        return $this->sessionId;
    }

    public function setSessionId(?string $sessionId): void
    {
        $this->sessionId = $sessionId;
    }

    public function getCurrentPlan(): ?object
    {
        return $this->currentPlan;
    }

    public function setCurrentPlan(?object $plan): void
    {
        $this->currentPlan = $plan;
    }

    public function sendNotification(string $method, array $params = []): void
    {
        if ($this->notificationCallback instanceof Closure) {
            ($this->notificationCallback)($method, $params);
        }
    }

    public function logToSession(string $message, string $type = 'info', array $data = []): void
    {
        if ($this->logCallback instanceof Closure) {
            ($this->logCallback)($message, $type, $data);
        }
    }

    public function setNotificationCallback(?Closure $callback): void
    {
        $this->notificationCallback = $callback;
    }

    public function setLogCallback(?Closure $callback): void
    {
        $this->logCallback = $callback;
    }

    public function hasSession(): bool
    {
        return $this->sessionId !== null;
    }

    public function hasPlan(): bool
    {
        return $this->currentPlan !== null;
    }
}
