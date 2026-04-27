<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Services\ShapeClassifier;

test('ShapeClassifier_classify_Good_returns_A_for_critical_urgent_and_escalation_tags', function (): void {
    $classifier = new ShapeClassifier;

    expect($classifier->classify([
        'severity' => 'critical',
        'priority' => 'normal',
        'tags' => [],
    ]))->toBe(ShapeClassifier::CLASS_A)
        ->and($classifier->classify((object) [
            'severity' => 'minor',
            'priority' => 'urgent',
            'tags' => [],
        ]))->toBe(ShapeClassifier::CLASS_A)
        ->and($classifier->classify([
            'severity' => 'minor',
            'priority' => 'low',
            'tags' => ['security', 'bugfix'],
        ]))->toBe(ShapeClassifier::CLASS_A);
});

test('ShapeClassifier_classify_Bad_returns_B_for_major_or_high_without_A_signals', function (): void {
    $classifier = new ShapeClassifier;

    expect($classifier->classify([
        'severity' => 'major',
        'priority' => 'normal',
        'tags' => ['dispatch'],
    ]))->toBe(ShapeClassifier::CLASS_B)
        ->and($classifier->classify((object) [
            'severity' => 'minor',
            'priority' => 'high',
            'tags' => ['review'],
        ]))->toBe(ShapeClassifier::CLASS_B);
});

test('ShapeClassifier_classify_Ugly_returns_C_when_no_escalation_signals_exist', function (): void {
    $classifier = new ShapeClassifier;

    expect($classifier->classify([
        'severity' => 'minor',
        'priority' => 'normal',
        'tags' => ['dispatch'],
    ]))->toBe(ShapeClassifier::CLASS_C)
        ->and($classifier->classify([
            'severity' => null,
            'priority' => null,
            'tags' => [],
        ]))->toBe(ShapeClassifier::CLASS_C);
});
