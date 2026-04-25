<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Documentation;

use Illuminate\Http\JsonResponse;

class DocumentationController
{
    public function index(): JsonResponse
    {
        return response()->json([
            'message' => 'Admin documentation tooling is reserved for the follow-up slice.',
        ], 501);
    }

    public function openApiJson(): JsonResponse
    {
        return response()->json([
            'message' => 'OpenAPI generation is not included in the foundation slice.',
        ], 501);
    }

    public function clearCache(): JsonResponse
    {
        return response()->json([
            'message' => 'Documentation cache clearing is not included in the foundation slice.',
        ], 501);
    }
}
