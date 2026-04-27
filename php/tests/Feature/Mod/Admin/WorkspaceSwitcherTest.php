<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Mod\Admin;

use Core\Mod\Agentic\Website\Hub\View\Modal\Admin\WorkspaceSwitcher;
use Core\Tenant\Services\WorkspaceService;
use Illuminate\Http\Request;
use Livewire\Livewire;

class WorkspaceSwitcherTest extends AdminTestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        $this->app->instance('request', Request::create('http://core.test/hub/workspaces'));
        $this->app->instance(WorkspaceService::class, new class extends WorkspaceService
        {
            protected string $slug = 'alpha';

            public function all(): array
            {
                return [
                    'alpha' => ['name' => 'Alpha', 'slug' => 'alpha', 'description' => 'Default workspace'],
                    'beta' => ['name' => 'Beta', 'slug' => 'beta', 'description' => 'Secondary workspace'],
                ];
            }

            public function current(): array
            {
                return $this->all()[$this->slug];
            }

            public function setCurrent(string $slug): bool
            {
                if (! isset($this->all()[$slug])) {
                    return false;
                }

                $this->slug = $slug;

                return true;
            }
        });
    }

    public function test_WorkspaceSwitcher_mount_Good_captures_return_url(): void
    {
        $this->actingAsHades();

        Livewire::test(WorkspaceSwitcher::class)
            ->assertSet('returnUrl', 'http://core.test/hub/workspaces')
            ->assertSet('current.slug', 'alpha');
    }

    public function test_WorkspaceSwitcher_switchWorkspace_Good_dispatches_event_and_redirects_back(): void
    {
        $this->actingAsHades();

        Livewire::test(WorkspaceSwitcher::class)
            ->call('switchWorkspace', 'beta')
            ->assertDispatched('workspace-changed', workspace: 'beta')
            ->assertRedirect('http://core.test/hub/workspaces');
    }
}
