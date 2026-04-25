<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\View\Components;

class Select extends FormComponent
{
    public string $id;

    public ?string $label;

    /**
     * @var array<string, string>
     */
    public array $options;

    public bool $multiple;

    public bool $required;

    public function __construct(
        string $id,
        ?string $label = null,
        array $options = [],
        bool $multiple = false,
        bool $required = false,
        bool $disabled = false,
        ?string $canGate = null,
        mixed $canResource = null,
        bool $canHide = false,
    ) {
        $this->id = $id;
        $this->label = $label;
        $this->options = $options;
        $this->multiple = $multiple;
        $this->required = $required;
        $this->initialiseAuthorisation($canGate, $canResource, $canHide, $disabled);
    }

    public function render()
    {
        return view('core-forms::components.forms.select');
    }
}
