<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Services/DataRedactor.php');

use Core\Mcp\Services\DataRedactor;

test('DataRedactor_redact_Good_fully_redacts_sensitive_keys_and_string_patterns', function (): void {
    $service = new DataRedactor;

    $redacted = $service->redact([
        'password' => 'super-secret',
        'header' => 'Bearer sk_1234567890abcdefghijklmn',
        'card' => '4111-1111-1111-1111',
    ]);

    expect($redacted['password'])->toBe('[REDACTED]')
        ->and($redacted['header'])->toContain('Bearer [REDACTED]')
        ->and($redacted['card'])->toBe('[REDACTED]');
});

test('DataRedactor_summarize_Bad_partially_redacts_pii_and_truncates_large_arrays', function (): void {
    $service = new DataRedactor;

    $summary = $service->summarize([
        'email' => 'somebody@example.com',
        'items' => range(1, 12),
    ]);

    expect($summary['email'])->toBe('som***com')
        ->and($summary['items']['_truncated'])->toBe('... and 2 more items');
});

test('DataRedactor_redact_Ugly_stops_when_the_maximum_depth_is_exceeded', function (): void {
    $service = new DataRedactor;

    $redacted = $service->redact([
        'level1' => [
            'level2' => [
                'level3' => ['secret' => 'value'],
            ],
        ],
    ], 2);

    expect($redacted['level1']['level2'])->toBe('[MAX_DEPTH_EXCEEDED]');
});
