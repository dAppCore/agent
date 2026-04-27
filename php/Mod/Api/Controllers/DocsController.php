<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Controllers;

use Illuminate\Http\JsonResponse;

class DocsController
{
    public function index(): JsonResponse
    {
        return response()->json([
            'message' => 'Public API documentation portal follows in the next RFC slice.',
        ], 501);
    }

    public function openapi(): JsonResponse
    {
        return response()->json([
            'message' => 'Public OpenAPI export follows in the next RFC slice.',
        ], 501);
    }
}
