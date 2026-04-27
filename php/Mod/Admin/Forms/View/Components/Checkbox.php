<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\View\Components;

class Checkbox extends FormComponent
{
    public string $id;

    public ?string $label;

    public bool $required;

    public function __construct(
        string $id,
        ?string $label = null,
        bool $required = false,
        bool $disabled = false,
        ?string $canGate = null,
        mixed $canResource = null,
        bool $canHide = false,
    ) {
        $this->id = $id;
        $this->label = $label;
        $this->required = $required;
        $this->initialiseAuthorisation($canGate, $canResource, $canHide, $disabled);
    }

    public function render()
    {
        return view('core-forms::components.forms.checkbox');
    }
}
