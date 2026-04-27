<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;

#[Title('Account Usage')]
#[Layout('hub::admin.layouts.app')]
class AccountUsage extends Component
{
    #[Url(as: 'tab')]
    public string $tab = 'overview';

    #[Computed]
    public function allowedTabs(): array
    {
        return ['overview', 'boosts', 'ai'];
    }

    #[Computed]
    public function stats(): array
    {
        $stats = auth()->user()?->cached_stats;

        if (! is_array($stats)) {
            return [
                'storage_used' => '0 GB',
                'api_calls' => 0,
                'seats' => 1,
                'boosts' => 0,
            ];
        }

        return array_replace([
            'storage_used' => '0 GB',
            'api_calls' => 0,
            'seats' => 1,
            'boosts' => 0,
        ], $stats);
    }

    public function mount(): void
    {
        if (! in_array($this->tab, $this->allowedTabs(), true)) {
            $this->tab = 'overview';
        }
    }

    public function render()
    {
        return view('hub::admin.account-usage');
    }
}
