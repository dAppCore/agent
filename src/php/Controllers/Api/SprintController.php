<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SprintController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        return response()->json(['data' => [], 'total' => 0]);
    }

    public function show(Request $request, string $slug): JsonResponse
    {
        return response()->json(['data' => null], 404);
    }

    public function store(Request $request): JsonResponse
    {
        return response()->json(['data' => null], 501);
    }

    public function update(Request $request, string $slug): JsonResponse
    {
        return response()->json(['data' => null], 501);
    }

    public function destroy(Request $request, string $slug): JsonResponse
    {
        return response()->json(['data' => null], 501);
    }
}
