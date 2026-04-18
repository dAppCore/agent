<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Tests\TestCase;

class AgenticGenerateCommandTest extends TestCase
{
    public function test_it_registers_the_canonical_agentic_generate_command(): void
    {
        $this->artisan('help', [
            'command' => 'agentic:generate',
        ])
            ->expectsOutputToContain('agentic:generate')
            ->assertSuccessful();
    }
}
