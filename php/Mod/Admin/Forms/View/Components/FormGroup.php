<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\View\Components;

class FormGroup extends FormComponent
{
    public ?string $label;

    public ?string $error;

    public ?string $help;

    public bool $required;

    public function __construct(
        ?string $label = null,
        ?string $error = null,
        ?string $help = null,
        bool $required = false,
        bool $disabled = false,
        ?string $canGate = null,
        mixed $canResource = null,
        bool $canHide = false,
    ) {
        $this->label = $label;
        $this->error = $error;
        $this->help = $help;
        $this->required = $required;
        $this->initialiseAuthorisation($canGate, $canResource, $canHide, $disabled);
    }

    public function render()
    {
        return view('core-forms::components.forms.form-group');
    }
}
