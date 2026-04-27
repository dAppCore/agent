<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Core\Mod\Agentic\Website\Hub\Concerns\HasRateLimiting;
use Core\Tenant\Enums\UserTier;
use Core\Tenant\Models\User;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Platform User')]
#[Layout('hub::admin.layouts.app')]
class PlatformUser extends Component
{
    use HasRateLimiting;

    public User $userRecord;

    public string $editingTier = 'free';

    public bool $editingVerified = false;

    public string $actionMessage = '';

    public string $actionType = '';

    public string $activeTab = 'overview';

    public function mount(int $id): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403);
        }

        $this->userRecord = User::query()->findOrFail($id);
        $this->editingTier = is_object($this->userRecord->tier)
            ? (string) $this->userRecord->tier->value
            : (string) ($this->userRecord->tier ?? 'free');
        $this->editingVerified = $this->userRecord->email_verified_at !== null;
    }

    public function setTab(string $tab): void
    {
        if (in_array($tab, ['overview', 'workspaces', 'security'], true)) {
            $this->activeTab = $tab;
        }
    }

    public function saveTier(): void
    {
        $this->rateLimit('tier-change', 10, function (): void {
            if (class_exists(UserTier::class)) {
                $this->userRecord->tier = UserTier::from($this->editingTier);
            } else {
                $this->userRecord->tier = $this->editingTier;
            }

            $this->userRecord->save();
            $this->actionMessage = sprintf('Tier updated to %s.', $this->editingTier);
            $this->actionType = 'success';
        });
    }

    public function saveVerification(): void
    {
        $this->rateLimit('verify-user', 10, function (): void {
            $this->userRecord->email_verified_at = $this->editingVerified ? now() : null;
            $this->userRecord->save();
            $this->actionMessage = $this->editingVerified ? 'Email marked as verified.' : 'Email verification removed.';
            $this->actionType = 'success';
        });
    }

    public function render()
    {
        return view('hub::admin.platform-user');
    }
}
