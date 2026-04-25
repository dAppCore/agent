<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Support\Facades\Schema;

test('agent foundation migrations expose the fleet task pointer and credit ledger columns', function (): void {
    expect(Schema::hasColumn('fleet_nodes', 'current_task_id'))->toBeTrue()
        ->and(Schema::hasColumn('credit_entries', 'agent_id'))->toBeTrue()
        ->and(Schema::hasColumn('credit_entries', 'fleet_task_id'))->toBeTrue();
});
