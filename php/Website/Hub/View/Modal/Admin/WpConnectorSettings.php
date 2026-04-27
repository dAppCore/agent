<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\View\Modal\Admin;

use Core\Tenant\Models\Workspace;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;
use Livewire\Attributes\Computed;
use Livewire\Component;

class WpConnectorSettings extends Component
{
    public Workspace $workspace;

    public bool $enabled = false;

    public string $wordpressUrl = '';

    public bool $testing = false;

    public ?string $testResult = null;

    public bool $testSuccess = false;

    public function mount(Workspace $workspace): void
    {
        $this->workspace = $workspace;
        $this->enabled = (bool) ($workspace->wp_connector_enabled ?? false);
        $this->wordpressUrl = (string) ($workspace->wp_connector_url ?? '');
    }

    #[Computed]
    public function webhookUrl(): string
    {
        return (string) ($this->workspace->wp_connector_webhook_url ?? '');
    }

    #[Computed]
    public function webhookSecret(): string
    {
        return (string) ($this->workspace->wp_connector_secret ?? '');
    }

    #[Computed]
    public function isVerified(): bool
    {
        return $this->workspace->wp_connector_verified_at !== null;
    }

    #[Computed]
    public function lastSync(): ?string
    {
        return $this->workspace->wp_connector_last_sync?->diffForHumans();
    }

    public function save(): void
    {
        $this->validate([
            'wordpressUrl' => ['nullable', 'url'],
        ]);

        if ($this->enabled && $this->wordpressUrl === '') {
            $this->addError('wordpressUrl', 'WordPress URL is required when the connector is enabled.');

            return;
        }

        if ($this->enabled && method_exists($this->workspace, 'enableWpConnector')) {
            $this->workspace->enableWpConnector($this->wordpressUrl);
        } elseif (! $this->enabled && method_exists($this->workspace, 'disableWpConnector')) {
            $this->workspace->disableWpConnector();
        } else {
            $this->workspace->wp_connector_enabled = $this->enabled;
            $this->workspace->wp_connector_url = $this->enabled ? $this->wordpressUrl : null;

            if ($this->enabled && empty($this->workspace->wp_connector_secret)) {
                $this->workspace->wp_connector_secret = Str::random(40);
            }

            $this->workspace->save();
        }

        $this->workspace->refresh();
        $this->dispatch('notify', message: 'WordPress connector updated.');
    }

    public function regenerateSecret(): void
    {
        if (method_exists($this->workspace, 'generateWpConnectorSecret')) {
            $this->workspace->generateWpConnectorSecret();
        } else {
            $this->workspace->wp_connector_secret = Str::random(40);
            $this->workspace->save();
        }

        $this->workspace->refresh();
        $this->dispatch('notify', message: 'Webhook secret regenerated.');
    }

    public function testConnection(): void
    {
        $this->testing = true;
        $this->testSuccess = false;
        $this->testResult = null;

        if ($this->wordpressUrl === '') {
            $this->testResult = 'WordPress URL is not configured.';
            $this->testing = false;

            return;
        }

        try {
            $response = Http::timeout(10)->get(rtrim($this->wordpressUrl, '/').'/wp-json/wp/v2');

            if ($response->successful()) {
                $this->testSuccess = true;
                $this->testResult = 'Connected to the WordPress REST API.';
            } else {
                $this->testResult = 'WordPress returned HTTP '.$response->status().'.';
            }
        } catch (\Throwable $throwable) {
            $this->testResult = 'Connection failed: '.$throwable->getMessage();
        }

        $this->testing = false;
    }

    public function render()
    {
        return view('hub::admin.wp-connector-settings');
    }
}
