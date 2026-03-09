<?php

declare(strict_types=1);

use Core\Mod\Agentic\Controllers\AgentApiController;
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

// Authenticated agent endpoints
Route::middleware(AgentApiAuth::class.':plans.read')->group(function () {
    // Plans (read)
    Route::get('v1/plans', [AgentApiController::class, 'listPlans']);
    Route::get('v1/plans/{slug}', [AgentApiController::class, 'getPlan']);

    // Phases (read)
    Route::get('v1/plans/{slug}/phases/{phase}', [AgentApiController::class, 'getPhase']);

    // Sessions (read)
    Route::get('v1/sessions', [AgentApiController::class, 'listSessions']);
    Route::get('v1/sessions/{sessionId}', [AgentApiController::class, 'getSession']);
});

Route::middleware(AgentApiAuth::class.':plans.write')->group(function () {
    // Plans (write)
    Route::post('v1/plans', [AgentApiController::class, 'createPlan']);
    Route::patch('v1/plans/{slug}', [AgentApiController::class, 'updatePlan']);
    Route::delete('v1/plans/{slug}', [AgentApiController::class, 'archivePlan']);
});

Route::middleware(AgentApiAuth::class.':phases.write')->group(function () {
    // Phases (write)
    Route::patch('v1/plans/{slug}/phases/{phase}', [AgentApiController::class, 'updatePhase']);
    Route::post('v1/plans/{slug}/phases/{phase}/checkpoint', [AgentApiController::class, 'addCheckpoint']);
    Route::patch('v1/plans/{slug}/phases/{phase}/tasks/{taskIdx}', [AgentApiController::class, 'updateTask'])
        ->whereNumber('taskIdx');
    Route::post('v1/plans/{slug}/phases/{phase}/tasks/{taskIdx}/toggle', [AgentApiController::class, 'toggleTask'])
        ->whereNumber('taskIdx');
});

Route::middleware(AgentApiAuth::class.':sessions.write')->group(function () {
    // Sessions (write)
    Route::post('v1/sessions', [AgentApiController::class, 'startSession']);
    Route::post('v1/sessions/{sessionId}/end', [AgentApiController::class, 'endSession']);
    Route::post('v1/sessions/{sessionId}/continue', [AgentApiController::class, 'continueSession']);
});
