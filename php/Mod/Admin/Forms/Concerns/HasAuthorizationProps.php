<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\Concerns;

trait HasAuthorizationProps
{
    public ?string $canGate = null;

    public mixed $canResource = null;

    public bool $canHide = false;

    protected function resolveDisabledState(bool $disabled = false): bool
    {
        if ($disabled) {
            return true;
        }

        if ($this->canGate === null || $this->canResource === null) {
            return false;
        }

        return ! $this->userCan();
    }

    protected function resolveHiddenState(): bool
    {
        if (! $this->canHide) {
            return false;
        }

        if ($this->canGate === null || $this->canResource === null) {
            return false;
        }

        return ! $this->userCan();
    }

    protected function userCan(): bool
    {
        $user = auth()->user();

        if ($user === null || ! method_exists($user, 'can')) {
            return false;
        }

        return (bool) $user->can($this->canGate, $this->canResource);
    }
}
