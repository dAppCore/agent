<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Core\Mod\Agentic\Website\Hub\Concerns\HasRateLimiting;
use Core\Tenant\Models\User;
use Illuminate\Pagination\LengthAwarePaginator;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;
use Livewire\WithPagination;

#[Title('Platform')]
#[Layout('hub::admin.layouts.app')]
class Platform extends Component
{
    use HasRateLimiting;
    use WithPagination;

    public string $search = '';

    public string $tierFilter = '';

    public string $verifiedFilter = '';

    public string $sortField = 'created_at';

    public string $sortDirection = 'desc';

    public string $actionMessage = '';

    public string $actionType = '';

    protected $queryString = [
        'search' => ['except' => ''],
        'tierFilter' => ['except' => ''],
        'verifiedFilter' => ['except' => ''],
    ];

    public function mount(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403);
        }
    }

    public function updatingSearch(): void
    {
        $this->resetPage();
    }

    public function sortBy(string $field): void
    {
        if ($this->sortField === $field) {
            $this->sortDirection = $this->sortDirection === 'asc' ? 'desc' : 'asc';

            return;
        }

        $this->sortField = $field;
        $this->sortDirection = 'asc';
    }

    public function verifyEmail(int $userId): void
    {
        $this->rateLimit('verify-email', 10, function () use ($userId): void {
            $user = class_exists(User::class) ? User::query()->find($userId) : null;

            if ($user === null || $user->email_verified_at !== null) {
                return;
            }

            if (method_exists($user, 'markEmailAsVerified')) {
                $user->markEmailAsVerified();
            } else {
                $user->email_verified_at = now();
                $user->save();
            }

            $this->actionMessage = sprintf('Email verified for %s.', $user->email);
            $this->actionType = 'success';
        });
    }

    public function render()
    {
        return view('hub::admin.platform', [
            'users' => $this->users(),
        ]);
    }

    protected function users(): LengthAwarePaginator
    {
        if (! class_exists(User::class)) {
            return new LengthAwarePaginator([], 0, 20, $this->getPage());
        }

        return User::query()
            ->when($this->search !== '', function ($query): void {
                $query->where(function ($inner): void {
                    $inner->where('name', 'like', '%'.$this->search.'%')
                        ->orWhere('email', 'like', '%'.$this->search.'%');
                });
            })
            ->when($this->tierFilter !== '', fn ($query) => $query->where('tier', $this->tierFilter))
            ->when($this->verifiedFilter !== '', function ($query): void {
                if ($this->verifiedFilter === 'verified') {
                    $query->whereNotNull('email_verified_at');
                } elseif ($this->verifiedFilter === 'unverified') {
                    $query->whereNull('email_verified_at');
                }
            })
            ->orderBy($this->sortField, $this->sortDirection)
            ->paginate(20);
    }
}
