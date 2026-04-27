<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\View\Components;

class Input extends FormComponent
{
    public string $id;

    public ?string $label;

    public ?string $hint;

    public string $type;

    public ?string $placeholder;

    public bool $required;

    public bool $readonly;

    public function __construct(
        string $id,
        ?string $label = null,
        ?string $hint = null,
        string $type = 'text',
        ?string $placeholder = null,
        bool $required = false,
        bool $disabled = false,
        bool $readonly = false,
        ?string $canGate = null,
        mixed $canResource = null,
        bool $canHide = false,
    ) {
        $this->id = $id;
        $this->label = $label;
        $this->hint = $hint;
        $this->type = $type;
        $this->placeholder = $placeholder;
        $this->required = $required;
        $this->readonly = $readonly;
        $this->initialiseAuthorisation($canGate, $canResource, $canHide, $disabled);
    }

    public function render()
    {
        return view('core-forms::components.forms.input');
    }
}
