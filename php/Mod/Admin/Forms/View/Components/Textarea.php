<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\View\Components;

class Textarea extends FormComponent
{
    public string $id;

    public ?string $label;

    public ?string $hint;

    public int $rows;

    public ?string $placeholder;

    public bool $required;

    public function __construct(
        string $id,
        ?string $label = null,
        ?string $hint = null,
        int $rows = 4,
        ?string $placeholder = null,
        bool $required = false,
        bool $disabled = false,
        ?string $canGate = null,
        mixed $canResource = null,
        bool $canHide = false,
    ) {
        $this->id = $id;
        $this->label = $label;
        $this->hint = $hint;
        $this->rows = $rows;
        $this->placeholder = $placeholder;
        $this->required = $required;
        $this->initialiseAuthorisation($canGate, $canResource, $canHide, $disabled);
    }

    public function render()
    {
        return view('core-forms::components.forms.textarea');
    }
}
