<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Mcp\Services;

use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainRemember;
use Core\Mod\Agentic\Services\AgentToolRegistry;
use Tests\TestCase;

/**
 * Guards the outage this suite could not see for as long as it existed.
 *
 * Three separate faults kept the agent MCP server serving nothing, and a green
 * suite never noticed any of them:
 *
 *   1. every tool class was fatal on load, because AgentTool used a trait this
 *      repo did not contain;
 *   2. Boot filled the registry from a $listens event that ModuleScanner only
 *      wires from app/Core|Mod|Website, so it never fired under vendor/;
 *   3. the stdio server read a second, never-bound registry class entirely.
 *
 * McpAgentServerCommandTest passed throughout, because its beforeEach
 * registered a tool of its own — it supplied exactly what production lacked.
 * So these assert on a plain booted application and register nothing.
 */
class AgentToolRegistryBootTest extends TestCase
{
    public function test_tools_good_construct_at_all(): void
    {
        // The floor. For months this line raised
        // Trait "Core\Mcp\Tools\Concerns\ValidatesDependencies" not found.
        $this->assertSame('brain_remember', (new BrainRemember)->name());
    }

    public function test_registry_good_is_filled_by_boot_with_no_test_supplied_tools(): void
    {
        $registry = $this->app->make(AgentToolRegistry::class);

        $this->assertNotEmpty(
            $registry->all(),
            'Boot registered no tools — the agent MCP server would advertise nothing.',
        );
    }

    public function test_registry_good_exposes_those_tools_on_the_path_the_server_reads(): void
    {
        // listTools() is what McpAgentServerCommand answers tools/list from.
        // Filled-but-unreadable was the shape of fault 3.
        $names = array_map(
            static fn ($tool): string => $tool->name,
            $this->app->make(AgentToolRegistry::class)->listTools(),
        );

        $this->assertNotEmpty($names);
        $this->assertContains('plan_create', $names);
        $this->assertContains('session_start', $names);
        $this->assertContains('brain_remember', $names);
    }

    public function test_registry_good_is_one_shared_instance(): void
    {
        // A non-singleton hands the command a fresh empty registry, which is
        // how the registry it used to read failed even once something filled it.
        $this->assertSame(
            $this->app->make(AgentToolRegistry::class),
            $this->app->make(AgentToolRegistry::class),
        );
    }
}
