<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\View\Modal\Admin\Sessions;
use Core\Tenant\Models\Workspace;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

class SessionsTest extends LivewireTestCase
{
    private Workspace $workspace;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
    }

    public function test_requires_hades_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(Sessions::class);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(Sessions::class)
            ->assertOk();
    }

    public function test_has_default_property_values(): void
    {
        $this->actingAsHades();

        Livewire::test(Sessions::class)
            ->assertSet('search', '')
            ->assertSet('status', '')
            ->assertSet('agentType', '')
            ->assertSet('workspace', '')
            ->assertSet('planSlug', '')
            ->assertSet('perPage', 20);
    }

    public function test_search_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(Sessions::class)
            ->set('search', 'session-abc')
            ->assertSet('search', 'session-abc');
    }

    public function test_status_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(Sessions::class)
            ->set('status', AgentSession::STATUS_ACTIVE)
            ->assertSet('status', AgentSession::STATUS_ACTIVE);
    }

    public function test_agent_type_filter_updates(): void
    {
        $this->actingAsHades();

        Livewire::test(Sessions::class)
            ->set('agentType', AgentSession::AGENT_SONNET)
            ->assertSet('agentType', AgentSession::AGENT_SONNET);
    }

    public function test_clear_filters_resets_all_filters(): void
    {
        $this->actingAsHades();

        Livewire::test(Sessions::class)
            ->set('search', 'test')
            ->set('status', AgentSession::STATUS_ACTIVE)
            ->set('agentType', AgentSession::AGENT_OPUS)
            ->set('workspace', '1')
            ->set('planSlug', 'some-plan')
            ->call('clearFilters')
            ->assertSet('search', '')
            ->assertSet('status', '')
            ->assertSet('agentType', '')
            ->assertSet('workspace', '')
            ->assertSet('planSlug', '');
    }

    public function test_pause_session_changes_status(): void
    {
        $this->actingAsHades();

        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        Livewire::test(Sessions::class)
            ->call('pause', $session->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentSession::STATUS_PAUSED, $session->fresh()->status);
    }

    public function test_resume_session_changes_status(): void
    {
        $this->actingAsHades();

        $session = AgentSession::factory()->paused()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        Livewire::test(Sessions::class)
            ->call('resume', $session->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentSession::STATUS_ACTIVE, $session->fresh()->status);
    }

    public function test_complete_session_changes_status(): void
    {
        $this->actingAsHades();

        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        Livewire::test(Sessions::class)
            ->call('complete', $session->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentSession::STATUS_COMPLETED, $session->fresh()->status);
    }

    public function test_fail_session_changes_status(): void
    {
        $this->actingAsHades();

        $session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        Livewire::test(Sessions::class)
            ->call('fail', $session->id)
            ->assertDispatched('notify');

        $this->assertEquals(AgentSession::STATUS_FAILED, $session->fresh()->status);
    }

    public function test_get_status_color_class_returns_green_for_active(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Sessions::class);

        $class = $component->instance()->getStatusColorClass(AgentSession::STATUS_ACTIVE);

        $this->assertStringContainsString('green', $class);
    }

    public function test_get_status_color_class_returns_red_for_failed(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Sessions::class);

        $class = $component->instance()->getStatusColorClass(AgentSession::STATUS_FAILED);

        $this->assertStringContainsString('red', $class);
    }

    public function test_get_agent_badge_class_returns_class_for_opus(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Sessions::class);

        $class = $component->instance()->getAgentBadgeClass(AgentSession::AGENT_OPUS);

        $this->assertNotEmpty($class);
        $this->assertStringContainsString('violet', $class);
    }

    public function test_status_options_contains_all_statuses(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(Sessions::class);
        $options = $component->instance()->statusOptions;

        $this->assertArrayHasKey(AgentSession::STATUS_ACTIVE, $options);
        $this->assertArrayHasKey(AgentSession::STATUS_PAUSED, $options);
        $this->assertArrayHasKey(AgentSession::STATUS_COMPLETED, $options);
        $this->assertArrayHasKey(AgentSession::STATUS_FAILED, $options);
    }
}
