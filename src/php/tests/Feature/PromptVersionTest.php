<?php

declare(strict_types=1);

use Core\Mod\Agentic\Models\Prompt;
use Core\Mod\Agentic\Models\PromptVersion;
use Core\Tenant\Models\User;

// =========================================================================
// Version Creation Tests
// =========================================================================

describe('version creation', function () {
    it('can be created with required attributes', function () {
        $prompt = Prompt::create([
            'name' => 'Test Prompt',
            'system_prompt' => 'You are a helpful assistant.',
            'user_template' => 'Answer this: {{{question}}}',
            'variables' => ['question'],
        ]);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
            'system_prompt' => 'You are a helpful assistant.',
            'user_template' => 'Answer this: {{{question}}}',
            'variables' => ['question'],
        ]);

        expect($version->id)->not->toBeNull()
            ->and($version->version)->toBe(1)
            ->and($version->prompt_id)->toBe($prompt->id);
    });

    it('casts variables as array', function () {
        $prompt = Prompt::create(['name' => 'Test Prompt']);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
            'variables' => ['topic', 'tone'],
        ]);

        expect($version->variables)
            ->toBeArray()
            ->toBe(['topic', 'tone']);
    });

    it('casts version as integer', function () {
        $prompt = Prompt::create(['name' => 'Test Prompt']);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 3,
        ]);

        expect($version->version)->toBeInt()->toBe(3);
    });

    it('can be created without optional fields', function () {
        $prompt = Prompt::create(['name' => 'Minimal Prompt']);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
        ]);

        expect($version->id)->not->toBeNull()
            ->and($version->system_prompt)->toBeNull()
            ->and($version->user_template)->toBeNull()
            ->and($version->created_by)->toBeNull();
    });
});

// =========================================================================
// Relationship Tests
// =========================================================================

describe('relationships', function () {
    it('belongs to a prompt', function () {
        $prompt = Prompt::create([
            'name' => 'Parent Prompt',
            'system_prompt' => 'System text.',
        ]);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
        ]);

        expect($version->prompt)->toBeInstanceOf(Prompt::class)
            ->and($version->prompt->id)->toBe($prompt->id)
            ->and($version->prompt->name)->toBe('Parent Prompt');
    });

    it('belongs to a creator user', function () {
        $user = User::factory()->create();
        $prompt = Prompt::create(['name' => 'Test Prompt']);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
            'created_by' => $user->id,
        ]);

        expect($version->creator)->toBeInstanceOf(User::class)
            ->and($version->creator->id)->toBe($user->id);
    });

    it('has null creator when created_by is null', function () {
        $prompt = Prompt::create(['name' => 'Test Prompt']);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
        ]);

        expect($version->creator)->toBeNull();
    });
});

// =========================================================================
// Restore Method Tests
// =========================================================================

describe('restore', function () {
    it('restores system_prompt and user_template to the parent prompt', function () {
        $prompt = Prompt::create([
            'name' => 'Test Prompt',
            'system_prompt' => 'Original system prompt.',
            'user_template' => 'Original template.',
        ]);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
            'system_prompt' => 'Versioned system prompt.',
            'user_template' => 'Versioned template.',
        ]);

        $prompt->update([
            'system_prompt' => 'Newer system prompt.',
            'user_template' => 'Newer template.',
        ]);

        $version->restore();

        $fresh = $prompt->fresh();
        expect($fresh->system_prompt)->toBe('Versioned system prompt.')
            ->and($fresh->user_template)->toBe('Versioned template.');
    });

    it('restores variables to the parent prompt', function () {
        $prompt = Prompt::create([
            'name' => 'Test Prompt',
            'variables' => ['topic'],
        ]);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
            'variables' => ['topic', 'tone'],
        ]);

        $prompt->update(['variables' => ['topic', 'tone', 'length']]);

        $version->restore();

        expect($prompt->fresh()->variables)->toBe(['topic', 'tone']);
    });

    it('returns the parent prompt instance after restore', function () {
        $prompt = Prompt::create([
            'name' => 'Test Prompt',
            'system_prompt' => 'Old.',
        ]);

        $version = PromptVersion::create([
            'prompt_id' => $prompt->id,
            'version' => 1,
            'system_prompt' => 'Versioned.',
        ]);

        $result = $version->restore();

        expect($result)->toBeInstanceOf(Prompt::class)
            ->and($result->id)->toBe($prompt->id);
    });
});

// =========================================================================
// Version History Tests
// =========================================================================

describe('version history', function () {
    it('prompt tracks multiple versions in descending order', function () {
        $prompt = Prompt::create([
            'name' => 'Evolving Prompt',
            'system_prompt' => 'v1.',
        ]);

        PromptVersion::create(['prompt_id' => $prompt->id, 'version' => 1, 'system_prompt' => 'v1.']);
        PromptVersion::create(['prompt_id' => $prompt->id, 'version' => 2, 'system_prompt' => 'v2.']);
        PromptVersion::create(['prompt_id' => $prompt->id, 'version' => 3, 'system_prompt' => 'v3.']);

        $versions = $prompt->versions()->get();

        expect($versions)->toHaveCount(3)
            ->and($versions->first()->version)->toBe(3)
            ->and($versions->last()->version)->toBe(1);
    });

    it('createVersion snapshots current prompt state', function () {
        $prompt = Prompt::create([
            'name' => 'Test Prompt',
            'system_prompt' => 'Original system prompt.',
            'user_template' => 'Original template.',
            'variables' => ['topic'],
        ]);

        $version = $prompt->createVersion();

        expect($version)->toBeInstanceOf(PromptVersion::class)
            ->and($version->version)->toBe(1)
            ->and($version->system_prompt)->toBe('Original system prompt.')
            ->and($version->user_template)->toBe('Original template.')
            ->and($version->variables)->toBe(['topic']);
    });

    it('createVersion increments version number', function () {
        $prompt = Prompt::create([
            'name' => 'Test Prompt',
            'system_prompt' => 'v1.',
        ]);

        $v1 = $prompt->createVersion();
        $prompt->update(['system_prompt' => 'v2.']);
        $v2 = $prompt->createVersion();

        expect($v1->version)->toBe(1)
            ->and($v2->version)->toBe(2);
    });

    it('createVersion records the creator user id', function () {
        $user = User::factory()->create();
        $prompt = Prompt::create([
            'name' => 'Test Prompt',
            'system_prompt' => 'System text.',
        ]);

        $version = $prompt->createVersion($user->id);

        expect($version->created_by)->toBe($user->id);
    });

    it('versions are scoped to their parent prompt', function () {
        $promptA = Prompt::create(['name' => 'Prompt A']);
        $promptB = Prompt::create(['name' => 'Prompt B']);

        PromptVersion::create(['prompt_id' => $promptA->id, 'version' => 1]);
        PromptVersion::create(['prompt_id' => $promptA->id, 'version' => 2]);
        PromptVersion::create(['prompt_id' => $promptB->id, 'version' => 1]);

        expect($promptA->versions()->count())->toBe(2)
            ->and($promptB->versions()->count())->toBe(1);
    });

    it('deleting prompt cascades to versions', function () {
        $prompt = Prompt::create(['name' => 'Test Prompt']);

        PromptVersion::create(['prompt_id' => $prompt->id, 'version' => 1]);
        PromptVersion::create(['prompt_id' => $prompt->id, 'version' => 2]);

        $promptId = $prompt->id;
        $prompt->delete();

        expect(PromptVersion::where('prompt_id', $promptId)->count())->toBe(0);
    });
});
