<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Hub Dashboard')]
#[Layout('hub::admin.layouts.app')]
class Dashboard extends Component
{
    public function render()
    {
        return view('hub::admin.dashboard');
    }
}
