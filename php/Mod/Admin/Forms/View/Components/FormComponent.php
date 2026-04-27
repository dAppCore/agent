<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\View\Components;

use Core\Mod\Agentic\Mod\Admin\Forms\Concerns\HasAuthorizationProps;
use Illuminate\View\Component;

abstract class FormComponent extends Component
{
    use HasAuthorizationProps;

    public bool $disabled = false;

    public bool $hidden = false;

    protected function initialiseAuthorisation(
        ?string $canGate,
        mixed $canResource,
        bool $canHide,
        bool $disabled = false
    ): void {
        $this->canGate = $canGate;
        $this->canResource = $canResource;
        $this->canHide = $canHide;
        $this->disabled = $this->resolveDisabledState($disabled);
        $this->hidden = $this->resolveHiddenState();
    }

    public function shouldRender(): bool
    {
        return ! $this->hidden;
    }
}
