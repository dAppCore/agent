<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\View\Modal\Admin;

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Core\Tenant\Models\Workspace;
use Livewire\Attributes\Layout;
use Livewire\Component;

/**
 * MCP API Key Manager.
 *
 * Allows workspace owners to create and manage API keys
 * for accessing MCP servers via HTTP API.
 */
#[Layout('hub::admin.layouts.app')]
class ApiKeyManager extends Component
{
    public Workspace $workspace;

    // Create form state
    public bool $showCreateModal = false;

    public string $newKeyName = '';

    public array $newKeyPermissions = [];

    public string $newKeyExpiry = 'never';

    // Show new key (only visible once after creation)
    public ?string $newPlainKey = null;

    public bool $showNewKeyModal = false;

    public function mount(Workspace $workspace): void
    {
        $this->workspace = $workspace;
    }

    public function openCreateModal(): void
    {
        $this->showCreateModal = true;
        $this->newKeyName = '';
        $this->newKeyPermissions = [];
        $this->newKeyExpiry = 'never';
    }

    public function closeCreateModal(): void
    {
        $this->showCreateModal = false;
    }

    public function availablePermissions(): array
    {
        return AgentApiKey::availablePermissions();
    }

    public function createKey(): void
    {
        $this->validate([
            'newKeyName' => 'required|string|max:100',
        ]);

        $expiresAt = match ($this->newKeyExpiry) {
            '30days' => now()->addDays(30),
            '90days' => now()->addDays(90),
            '1year' => now()->addYear(),
            default => null,
        };

        $key = app(AgentApiKeyService::class)->create(
            workspace: $this->workspace,
            name: $this->newKeyName,
            permissions: $this->newKeyPermissions,
            expiresAt: $expiresAt,
        );

        $this->newPlainKey = $key->plainTextKey;
        $this->showCreateModal = false;
        $this->showNewKeyModal = true;

        session()->flash('message', 'API key created successfully.');
    }

    public function closeNewKeyModal(): void
    {
        $this->newPlainKey = null;
        $this->showNewKeyModal = false;
    }

    public function revokeKey(int $keyId): void
    {
        $key = AgentApiKey::forWorkspace($this->workspace)->findOrFail($keyId);
        $key->revoke();

        session()->flash('message', 'API key revoked.');
    }

    public function togglePermission(string $permission): void
    {
        if (in_array($permission, $this->newKeyPermissions)) {
            $this->newKeyPermissions = array_values(array_diff($this->newKeyPermissions, [$permission]));
        } else {
            $this->newKeyPermissions[] = $permission;
        }
    }

    public function render()
    {
        return view('agentic::admin.api-key-manager', [
            'keys' => AgentApiKey::forWorkspace($this->workspace)->orderByDesc('created_at')->get(),
        ]);
    }
}
