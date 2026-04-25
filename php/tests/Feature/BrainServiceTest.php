<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature;

use Core\Mod\Agentic\Services\BrainService;
use PHPUnit\Framework\TestCase;

class BrainServiceTest extends TestCase
{
    public function test_BrainService_buildQdrantFilter_Good_FiltersByOrgOnly(): void
    {
        $filter = (new BrainService)->buildQdrantFilter([
            'org' => 'core',
        ]);

        $this->assertSame([
            'must' => [
                ['key' => 'org', 'match' => ['value' => 'core']],
            ],
        ], $filter);
    }

    public function test_BrainService_buildQdrantFilter_Bad_CombinesOrgAndProject(): void
    {
        $filter = (new BrainService)->buildQdrantFilter([
            'org' => 'core',
            'project' => 'agent',
        ]);

        $this->assertSame([
            'must' => [
                ['key' => 'org', 'match' => ['value' => 'core']],
                ['key' => 'project', 'match' => ['value' => 'agent']],
            ],
        ], $filter);
    }

    public function test_BrainService_buildQdrantFilter_Ugly_CombinesWorkspaceOrgAndProject(): void
    {
        $filter = (new BrainService)->buildQdrantFilter([
            'workspace_id' => 42,
            'org' => 'core',
            'project' => 'agent',
        ]);

        $this->assertSame([
            'must' => [
                ['key' => 'workspace_id', 'match' => ['value' => 42]],
                ['key' => 'org', 'match' => ['value' => 'core']],
                ['key' => 'project', 'match' => ['value' => 'agent']],
            ],
        ], $filter);
    }
}
