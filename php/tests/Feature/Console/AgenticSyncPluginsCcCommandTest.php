<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Console\Commands\AgenticSyncPluginsCcCommand;
use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Support\Facades\File;

beforeEach(function (): void {
    if (! is_string(config('app.key')) || config('app.key') === '') {
        config(['app.key' => 'base64:'.base64_encode(random_bytes(32))]);
    }

    $this->originalHome = getenv('HOME') ?: null;
    $this->temporaryHome = sys_get_temp_dir().'/agentic-sync-plugins-cc-'.uniqid('', true);

    File::ensureDirectoryExists($this->temporaryHome);

    putenv("HOME={$this->temporaryHome}");
    $_SERVER['HOME'] = $this->temporaryHome;
    $_ENV['HOME'] = $this->temporaryHome;

    $this->app->make(Kernel::class)->registerCommand(
        $this->app->make(AgenticSyncPluginsCcCommand::class),
    );
});

afterEach(function (): void {
    if (is_string($this->originalHome) && $this->originalHome !== '') {
        putenv("HOME={$this->originalHome}");
        $_SERVER['HOME'] = $this->originalHome;
        $_ENV['HOME'] = $this->originalHome;
    } else {
        putenv('HOME');
        unset($_SERVER['HOME'], $_ENV['HOME']);
    }

    if (isset($this->temporaryHome) && is_string($this->temporaryHome) && File::isDirectory($this->temporaryHome)) {
        File::deleteDirectory($this->temporaryHome);
    }
});

function agenticSyncPluginsCcProfile(array $attributes = []): AgentProfile
{
    return AgentProfile::create(array_merge([
        'name' => $attributes['name'] ?? 'dispatch-profile',
        'plugin_cc_name' => $attributes['plugin_cc_name'] ?? null,
        'gateway_url' => 'https://gateway.example.com',
        'api_key_cipher' => 'plain-secret',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
    ], $attributes));
}

function createClaudePluginDirectory(string $home, string $pluginName): void
{
    $pluginPath = $home.'/.claude/plugins/'.$pluginName;

    File::ensureDirectoryExists($pluginPath);
    File::put(
        $pluginPath.'/plugin.json',
        json_encode(['name' => $pluginName], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES) ?: '{}',
    );
}

test('AgenticSyncPluginsCcCommand_handle_Good_maps_plugins_to_enabled_profiles_and_reports_unmapped_entries', function (): void {
    $nameMatchedProfile = agenticSyncPluginsCcProfile([
        'name' => 'claude-ops',
    ]);
    $aliasMatchedProfile = agenticSyncPluginsCcProfile([
        'name' => 'dispatch-fleet',
        'plugin_cc_name' => 'legacy-bridge',
    ]);
    $disabledProfile = agenticSyncPluginsCcProfile([
        'name' => 'disabled-plugin',
        'enabled' => false,
    ]);
    $unmatchedProfile = agenticSyncPluginsCcProfile([
        'name' => 'spare-profile',
    ]);

    createClaudePluginDirectory($this->temporaryHome, 'claude-ops');
    createClaudePluginDirectory($this->temporaryHome, 'disabled-plugin');
    createClaudePluginDirectory($this->temporaryHome, 'legacy-bridge');
    createClaudePluginDirectory($this->temporaryHome, 'orphan-plugin');

    File::ensureDirectoryExists($this->temporaryHome.'/.claude/plugins/cache');
    File::ensureDirectoryExists($this->temporaryHome.'/.claude/plugins/data');

    $this->artisan('agentic:sync-plugins-cc')
        ->expectsTable(
            ['Plugin', 'Status', 'Profile', 'Match', 'Action'],
            [
                ['claude-ops', 'mapped', 'claude-ops', 'name', 'updated'],
                ['disabled-plugin', 'unmapped', '-', '-', 'no enabled profile'],
                ['legacy-bridge', 'mapped', 'dispatch-fleet', 'plugin_cc_name', 'unchanged'],
                ['orphan-plugin', 'unmapped', '-', '-', 'no enabled profile'],
            ],
        )
        ->expectsOutput('Mapped 2 Claude Code plugin(s); 2 unmapped.')
        ->assertSuccessful();

    expect($nameMatchedProfile->fresh()->plugin_cc_name)->toBe('claude-ops')
        ->and($aliasMatchedProfile->fresh()->plugin_cc_name)->toBe('legacy-bridge')
        ->and($disabledProfile->fresh()->plugin_cc_name)->toBeNull()
        ->and($unmatchedProfile->fresh()->plugin_cc_name)->toBeNull();
});

test('AgenticSyncPluginsCcCommand_handle_Ugly_reports_when_no_plugin_directories_exist', function (): void {
    $this->artisan('agentic:sync-plugins-cc')
        ->expectsOutput("No Claude Code plugin directories found in {$this->temporaryHome}/.claude/plugins.")
        ->assertSuccessful();
});
