<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Services\AgentDetection;
use Core\Mod\Agentic\Support\AgentIdentity;
use Illuminate\Http\Request;

/*
|--------------------------------------------------------------------------
| AgentDetection — the surface this repo owned but never tested
|--------------------------------------------------------------------------
|
| dappcore/php carried an AgentDetectionTest in its Module suite that tested
| classes this package owns. It could never pass there: php cannot depend on
| agent, because agent already depends on php, and the file named the classes
| under Core\Agentic\* — a namespace the ecosystem left behind — so it was
| wrong twice over.
|
| Twenty-two of its twenty-five cases duplicate what AgentDetectionTest here
| already covers, so importing the file wholesale would have added duplicate
| coverage to the repo that owns the code. Three concepts it covered were
| genuinely untested here, and only those are adopted:
|
|   AgentIdentity::getReferralPath()
|   AgentIdentity::getProviderDisplayName() / getModelDisplayName()
|   AgentDetection::identify() reading the X-MCP-Token header
|
| Written against this repo's real namespaces and API rather than ported.
|
*/

beforeEach(function (): void {
    $this->detection = new AgentDetection;
});

it('identifies a structured agent from the X-MCP-Token header', function (): void {
    $request = Request::create('/', 'GET', [], [], [], [
        'HTTP_X_MCP_TOKEN' => 'anthropic:claude-opus:secret123',
    ]);

    $identity = $this->detection->identify($request);

    expect($identity->isAgent())->toBeTrue()
        ->and($identity->provider)->toBe('anthropic')
        ->and($identity->model)->toBe('claude-opus')
        ->and($identity->confidence)->toBe('high');
});

it('treats an opaque X-MCP-Token as an agent of unknown provider', function (): void {
    // A registered agent presenting a bearer-style token still identifies as an
    // agent — it just cannot say which one.
    $request = Request::create('/', 'GET', [], [], [], [
        'HTTP_X_MCP_TOKEN' => 'some-opaque-token',
    ]);

    $identity = $this->detection->identify($request);

    expect($identity->isAgent())->toBeTrue()
        ->and($identity->provider)->toBe('unknown');
});

it('builds a referral path from provider and model', function (): void {
    expect((new AgentIdentity('anthropic', 'claude-opus', 'high'))->getReferralPath())
        ->toBe('/ref/anthropic/claude-opus');
});

it('builds a referral path without a model when none is known', function (): void {
    expect((new AgentIdentity('anthropic', null, 'high'))->getReferralPath())
        ->toBe('/ref/anthropic');
});

it('has no referral path for a request that is not an agent', function (): void {
    // The one branch php's file never reached: a non-agent must not be handed a
    // referral path at all.
    expect(AgentIdentity::notAnAgent()->getReferralPath())->toBeNull();
});

it('gives providers and models their display names', function (): void {
    $anthropic = new AgentIdentity('anthropic', 'claude-opus', 'high');
    $openai = new AgentIdentity('openai', 'gpt-4', 'high');

    expect($anthropic->getProviderDisplayName())->toBe('Anthropic')
        ->and($anthropic->getModelDisplayName())->toBe('Claude Opus')
        ->and($openai->getProviderDisplayName())->toBe('OpenAI')
        ->and($openai->getModelDisplayName())->toBe('GPT-4');
});

it('has no model display name when no model was detected', function (): void {
    expect(AgentIdentity::unknownAgent()->getModelDisplayName())->toBeNull();
});
