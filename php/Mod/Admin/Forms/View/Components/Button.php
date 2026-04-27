<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Forms\View\Components;

class Button extends FormComponent
{
    public string $type;

    public string $variant;

    public string $size;

    public ?string $icon;

    public bool $loading;

    public ?string $loadingText;

    public string $variantClasses;

    public string $sizeClasses;

    public function __construct(
        string $type = 'button',
        string $variant = 'primary',
        string $size = 'md',
        ?string $icon = null,
        bool $loading = false,
        ?string $loadingText = null,
        bool $disabled = false,
        ?string $canGate = null,
        mixed $canResource = null,
        bool $canHide = false,
    ) {
        $this->type = $type;
        $this->variant = $variant;
        $this->size = $size;
        $this->icon = $icon;
        $this->loading = $loading;
        $this->loadingText = $loadingText;
        $this->initialiseAuthorisation($canGate, $canResource, $canHide, $disabled);
        $this->variantClasses = $this->resolveVariantClasses();
        $this->sizeClasses = $this->resolveSizeClasses();
    }

    public function render()
    {
        return view('core-forms::components.forms.button');
    }

    protected function resolveVariantClasses(): string
    {
        return match ($this->variant) {
            'secondary' => 'bg-zinc-100 text-zinc-900 hover:bg-zinc-200',
            'danger' => 'bg-red-600 text-white hover:bg-red-700',
            'ghost' => 'bg-transparent text-zinc-700 hover:bg-zinc-100',
            default => 'bg-violet-600 text-white hover:bg-violet-700',
        };
    }

    protected function resolveSizeClasses(): string
    {
        return match ($this->size) {
            'sm' => 'px-3 py-1.5 text-sm',
            'lg' => 'px-5 py-3 text-base',
            default => 'px-4 py-2 text-sm',
        };
    }
}
