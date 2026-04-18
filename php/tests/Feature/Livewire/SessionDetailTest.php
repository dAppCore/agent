<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Livewire;

use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\View\Modal\Admin\SessionDetail;
use Core\Tenant\Models\Workspace;
use Livewire\Livewire;
use Symfony\Component\HttpKernel\Exception\HttpException;

class SessionDetailTest extends LivewireTestCase
{
    private Workspace $workspace;

    private AgentSession $session;

    protected function setUp(): void
    {
        parent::setUp();
        $this->workspace = Workspace::factory()->create();
        $this->session = AgentSession::factory()->active()->create([
            'workspace_id' => $this->workspace->id,
        ]);
    }

    public function test_requires_hades_access(): void
    {
        $this->expectException(HttpException::class);

        Livewire::test(SessionDetail::class, ['id' => $this->session->id]);
    }

    public function test_renders_successfully_with_hades_user(): void
    {
        $this->actingAsHades();

        Livewire::test(SessionDetail::class, ['id' => $this->session->id])
            ->assertOk();
    }

    public function test_mount_loads_session_by_id(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(SessionDetail::class, ['id' => $this->session->id]);

        $this->assertEquals($this->session->id, $component->instance()->session->id);
    }

    public function test_active_session_has_polling_enabled(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(SessionDetail::class, ['id' => $this->session->id]);

        $this->assertGreaterThan(0, $component->instance()->pollingInterval);
    }

    public function test_completed_session_disables_polling(): void
    {
        $this->actingAsHades();

        $completedSession = AgentSession::factory()->completed()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        $component = Livewire::test(SessionDetail::class, ['id' => $completedSession->id]);

        $this->assertEquals(0, $component->instance()->pollingInterval);
    }

    public function test_has_default_modal_states(): void
    {
        $this->actingAsHades();

        Livewire::test(SessionDetail::class, ['id' => $this->session->id])
            ->assertSet('showCompleteModal', false)
            ->assertSet('showFailModal', false)
            ->assertSet('showReplayModal', false)
            ->assertSet('completeSummary', '')
            ->assertSet('failReason', '');
    }

    public function test_pause_session_changes_status(): void
    {
        $this->actingAsHades();

        Livewire::test(SessionDetail::class, ['id' => $this->session->id])
            ->call('pauseSession')
            ->assertOk();

        $this->assertEquals(AgentSession::STATUS_PAUSED, $this->session->fresh()->status);
    }

    public function test_resume_session_changes_status_from_paused(): void
    {
        $this->actingAsHades();

        $pausedSession = AgentSession::factory()->paused()->create([
            'workspace_id' => $this->workspace->id,
        ]);

        Livewire::test(SessionDetail::class, ['id' => $pausedSession->id])
            ->call('resumeSession')
            ->assertOk();

        $this->assertEquals(AgentSession::STATUS_ACTIVE, $pausedSession->fresh()->status);
    }

    public function test_open_complete_modal_shows_modal(): void
    {
        $this->actingAsHades();

        Livewire::test(SessionDetail::class, ['id' => $this->session->id])
            ->call('openCompleteModal')
            ->assertSet('showCompleteModal', true);
    }

    public function test_open_fail_modal_shows_modal(): void
    {
        $this->actingAsHades();

        Livewire::test(SessionDetail::class, ['id' => $this->session->id])
            ->call('openFailModal')
            ->assertSet('showFailModal', true);
    }

    public function test_open_replay_modal_shows_modal(): void
    {
        $this->actingAsHades();

        Livewire::test(SessionDetail::class, ['id' => $this->session->id])
            ->call('openReplayModal')
            ->assertSet('showReplayModal', true);
    }

    public function test_work_log_returns_array(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(SessionDetail::class, ['id' => $this->session->id]);

        $this->assertIsArray($component->instance()->workLog);
    }

    public function test_artifacts_returns_array(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(SessionDetail::class, ['id' => $this->session->id]);

        $this->assertIsArray($component->instance()->artifacts);
    }

    public function test_get_status_color_class_returns_string(): void
    {
        $this->actingAsHades();

        $component = Livewire::test(SessionDetail::class, ['id' => $this->session->id]);

        $class = $component->instance()->getStatusColorClass(AgentSession::STATUS_ACTIVE);

        $this->assertNotEmpty($class);
    }
}
