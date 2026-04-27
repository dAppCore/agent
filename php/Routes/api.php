<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Controllers\AgentApiController;
use Core\Mod\Agentic\Controllers\Api\AuthController;
use Core\Mod\Agentic\Controllers\Api\BrainController;
use Core\Mod\Agentic\Controllers\Api\CreditsController;
use Core\Mod\Agentic\Controllers\Api\FleetController;
use Core\Mod\Agentic\Controllers\Api\IssueController;
use Core\Mod\Agentic\Controllers\Api\MessageController;
use Core\Mod\Agentic\Controllers\Api\PhaseController;
use Core\Mod\Agentic\Controllers\Api\PlanController;
use Core\Mod\Agentic\Controllers\Api\SessionController;
use Core\Mod\Agentic\Controllers\Api\SprintController;
use Core\Mod\Agentic\Controllers\Api\SubscriptionController;
use Core\Mod\Agentic\Controllers\Api\SyncController;
use Core\Mod\Agentic\Controllers\Api\TaskController;
use Core\Mod\Agentic\Middleware\AgentApiAuth;
use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| Agent API Routes
|--------------------------------------------------------------------------
|
| REST endpoints for the go-agentic Client (dispatch watch).
| Protected by AgentApiAuth middleware with Bearer token.
|
| Routes at /v1/* (Go client uses BaseURL + "/v1/...")
|
*/

// Health check (no auth required)
Route::get('v1/health', [AgentApiController::class, 'health']);

// GitHub App webhook (signature-verified, no Bearer auth)
Route::post('github/webhook', [\Core\Mod\Agentic\Controllers\Api\GitHubWebhookController::class, 'receive'])
    ->middleware('throttle:120,1');

Route::post('agentic/mantis-webhook', [\Core\Mod\Agentic\Http\Controllers\Api\MantisWebhookController::class, 'receive']);

// Agent checkin — discover which repos changed since last sync
// Uses auth.api (brain key) for authentication
Route::middleware(['throttle:120,1', 'auth.api:brain:read'])->group(function () {
    Route::get('v1/agent/checkin', [\Core\Mod\Agentic\Controllers\Api\CheckinController::class, 'checkin']);
});

Route::middleware(AgentApiAuth::class.':brain.read')->group(function () {
    Route::post('v1/brain/recall', [BrainController::class, 'recall']);
    Route::get('v1/brain/search', [BrainController::class, 'search']);
    Route::get('v1/brain/list', [BrainController::class, 'list']);
    Route::get('v1/brain/tags', [BrainController::class, 'tags']);
    Route::get('v1/brain/scopes', [BrainController::class, 'scopes']);
});

Route::middleware(AgentApiAuth::class.':brain.write')->group(function () {
    Route::post('v1/brain/remember', [BrainController::class, 'remember']);
    Route::delete('v1/brain/forget/{id}', [BrainController::class, 'forget']);
});

Route::middleware(AgentApiAuth::class.':plans.read')->group(function () {
    Route::get('v1/plans', [PlanController::class, 'index']);
    Route::get('v1/plans/{slug}', [PlanController::class, 'show']);
    Route::get('v1/plans/{slug}/phases/{phase}', [PhaseController::class, 'show']);
});

Route::middleware(AgentApiAuth::class.':plans.write')->group(function () {
    Route::post('v1/plans', [PlanController::class, 'store']);
    Route::patch('v1/plans/{slug}/status', [PlanController::class, 'update']);
    Route::delete('v1/plans/{slug}', [PlanController::class, 'destroy']);
});

Route::middleware(AgentApiAuth::class.':phases.write')->group(function () {
    Route::patch('v1/plans/{slug}/phases/{phase}', [PhaseController::class, 'update']);
    Route::post('v1/plans/{slug}/phases/{phase}/checkpoint', [PhaseController::class, 'checkpoint']);
    Route::patch('v1/plans/{slug}/phases/{phase}/tasks/{index}', [TaskController::class, 'update'])
        ->whereNumber('index');
    Route::post('v1/plans/{slug}/phases/{phase}/tasks/{index}/toggle', [TaskController::class, 'toggle'])
        ->whereNumber('index');
});

Route::middleware(AgentApiAuth::class.':sessions.read')->group(function () {
    Route::get('v1/sessions', [SessionController::class, 'index']);
    Route::get('v1/sessions/{id}', [SessionController::class, 'show']);
});

Route::middleware(AgentApiAuth::class.':sessions.write')->group(function () {
    Route::post('v1/sessions', [SessionController::class, 'store']);
    Route::post('v1/sessions/{id}/continue', [SessionController::class, 'continue']);
    Route::post('v1/sessions/{id}/end', [SessionController::class, 'end']);
});

// Issue tracker
Route::middleware(AgentApiAuth::class.':issues.read')->group(function () {
    Route::get('v1/issues', [IssueController::class, 'index']);
    Route::get('v1/issues/{slug}', [IssueController::class, 'show']);
    Route::get('v1/issues/{slug}/comments', [IssueController::class, 'comments']);
});

Route::middleware(AgentApiAuth::class.':issues.write')->group(function () {
    Route::post('v1/issues', [IssueController::class, 'store']);
    Route::patch('v1/issues/{slug}', [IssueController::class, 'update']);
    Route::delete('v1/issues/{slug}', [IssueController::class, 'destroy']);
    Route::post('v1/issues/{slug}/comments', [IssueController::class, 'addComment']);
});

// Sprints
Route::middleware(AgentApiAuth::class.':sprints.read')->group(function () {
    Route::get('v1/sprints', [SprintController::class, 'index']);
    Route::get('v1/sprints/{slug}', [SprintController::class, 'show']);
});

Route::middleware(AgentApiAuth::class.':sprints.write')->group(function () {
    Route::post('v1/sprints', [SprintController::class, 'store']);
    Route::patch('v1/sprints/{slug}', [SprintController::class, 'update']);
    Route::delete('v1/sprints/{slug}', [SprintController::class, 'destroy']);
});

Route::middleware(AgentApiAuth::class.':messages.read')->group(function () {
    Route::get('v1/messages/inbox', [MessageController::class, 'inbox']);
    Route::get('v1/messages/conversation/{agent}', [MessageController::class, 'conversation']);
});

Route::middleware(AgentApiAuth::class.':messages.write')->group(function () {
    Route::post('v1/messages/send', [MessageController::class, 'send']);
    Route::post('v1/messages/{id}/read', [MessageController::class, 'markRead']);
});

Route::middleware('auth')->group(function () {
    Route::post('v1/agent/auth/provision', [AuthController::class, 'provision']);
});

Route::middleware(AgentApiAuth::class.':auth.write')->group(function () {
    Route::delete('v1/agent/auth/revoke/{keyId}', [AuthController::class, 'revoke']);
});

Route::middleware(AgentApiAuth::class.':fleet.write')->group(function () {
    Route::post('v1/fleet/register', [FleetController::class, 'register']);
    Route::post('v1/fleet/heartbeat', [FleetController::class, 'heartbeat']);
    Route::post('v1/fleet/deregister', [FleetController::class, 'deregister']);
    Route::post('v1/fleet/task/assign', [FleetController::class, 'assignTask']);
    Route::post('v1/fleet/task/complete', [FleetController::class, 'completeTask']);
});

Route::middleware(AgentApiAuth::class.':fleet.read')->group(function () {
    Route::get('v1/fleet/nodes', [FleetController::class, 'index']);
    Route::get('v1/fleet/task/next', [FleetController::class, 'nextTask']);
    Route::get('v1/fleet/events', [FleetController::class, 'events']);
    Route::get('v1/fleet/stats', [FleetController::class, 'stats']);
});

Route::middleware(AgentApiAuth::class.':sync.write')->group(function () {
    Route::post('v1/agent/sync', [SyncController::class, 'push']);
});

Route::middleware(AgentApiAuth::class.':sync.read')->group(function () {
    Route::get('v1/agent/context', [SyncController::class, 'pull']);
    Route::get('v1/agent/status', [SyncController::class, 'status']);
});

Route::middleware(AgentApiAuth::class.':credits.write')->group(function () {
    Route::post('v1/credits/award', [CreditsController::class, 'award']);
});

Route::middleware(AgentApiAuth::class.':credits.read')->group(function () {
    Route::get('v1/credits/balance/{agentId}', [CreditsController::class, 'balance']);
    Route::get('v1/credits/history/{agentId}', [CreditsController::class, 'history']);
});

Route::middleware(AgentApiAuth::class.':subscription.write')->group(function () {
    Route::post('v1/subscription/detect', [SubscriptionController::class, 'detect']);
    Route::put('v1/subscription/budget/{agentId}', [SubscriptionController::class, 'updateBudget']);
});

Route::middleware(AgentApiAuth::class.':subscription.read')->group(function () {
    Route::get('v1/subscription/budget/{agentId}', [SubscriptionController::class, 'budget']);
});

Route::middleware(AgentApiAuth::class.':auth.write,sessions.write')->group(function () {
    Route::post('v1/agent/auth/register', [\Core\Mod\Agentic\Controllers\Api\AgentAuth\AgentAuthController::class, 'register']);
});

Route::middleware(AgentApiAuth::class.':fleet.write')->group(function () {
    Route::post('v1/fleet/dispatch', [\Core\Mod\Agentic\Controllers\Api\Fleet\FleetController::class, 'dispatch']);
});

Route::middleware(AgentApiAuth::class.':fleet.read')->group(function () {
    Route::get('v1/fleet/stream', [\Core\Mod\Agentic\Controllers\Api\Fleet\FleetController::class, 'stream']);
});

Route::middleware(AgentApiAuth::class.':credits.write')->group(function () {
    Route::post('v1/credits/deduct', [\Core\Mod\Agentic\Controllers\Api\Credits\CreditsController::class, 'deduct']);
    Route::post('v1/credits/refund', [\Core\Mod\Agentic\Controllers\Api\Credits\CreditsController::class, 'refund']);
});

Route::middleware(AgentApiAuth::class.':credits.read')->group(function () {
    Route::get('v1/credits/balance', [\Core\Mod\Agentic\Controllers\Api\Credits\CreditsController::class, 'balance']);
    Route::get('v1/credits/ledger', [\Core\Mod\Agentic\Controllers\Api\Credits\CreditsController::class, 'ledger']);
});

Route::middleware(AgentApiAuth::class.':subscription.write')->group(function () {
    Route::post('v1/subscription/upgrade', [\Core\Mod\Agentic\Controllers\Api\Subscription\SubscriptionController::class, 'upgrade']);
    Route::post('v1/subscription/cancel', [\Core\Mod\Agentic\Controllers\Api\Subscription\SubscriptionController::class, 'cancel']);
});

Route::middleware(AgentApiAuth::class.':subscription.read')->group(function () {
    Route::get('v1/subscription/status', [\Core\Mod\Agentic\Controllers\Api\Subscription\SubscriptionController::class, 'status']);
});

Route::middleware(AgentApiAuth::class.':sync.write')->group(function () {
    Route::post('v1/agent/sync/push', [\Core\Mod\Agentic\Controllers\Api\Sync\SyncController::class, 'push']);
});

Route::middleware(AgentApiAuth::class.':sync.read')->group(function () {
    Route::get('v1/agent/sync/pull', [\Core\Mod\Agentic\Controllers\Api\Sync\SyncController::class, 'pull']);
});
