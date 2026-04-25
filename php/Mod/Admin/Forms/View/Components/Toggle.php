<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\View\Components;

class Toggle extends FormComponent
{
    public string $id;

    public ?string $label;

    public function __construct(
        string $id,
        ?string $label = null,
        bool $disabled = false,
        ?string $canGate = null,
        mixed $canResource = null,
        bool $canHide = false,
    ) {
        $this->id = $id;
        $this->label = $label;
        $this->initialiseAuthorisation($canGate, $canResource, $canHide, $disabled);
    }

    public function render()
    {
        return view('core-forms::components.forms.toggle');
    }
}
