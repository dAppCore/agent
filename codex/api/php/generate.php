<?php

/**
 * This script parses a Laravel routes file and outputs a JSON representation of the
 * routes. It is designed to be used by the generate.sh script to generate an
 * API client.
 */
class ApiGenerator
{
    /**
     * A map of API resource actions to their corresponding client method names.
     * This is used to generate more user-friendly method names in the client.
     */
    private $actionMap = [
        'index' => 'list',
        'store' => 'create',
        'show' => 'get',
        'update' => 'update',
        'destroy' => 'delete',
    ];

    /**
     * The main method that parses the routes file and outputs the JSON.
     */
    public function generate()
    {
        // The path to the routes file.
        $routesFile = __DIR__ . '/routes/api.php';
        // The contents of the routes file.
        $contents = file_get_contents($routesFile);

        // An array to store the parsed routes.
        $output = [];

        // This regex matches Route::apiResource() declarations. It captures the
        // resource name (e.g., "users") and the controller name (e.g., "UserController").
        preg_match_all('/Route::apiResource\(\s*\'([^\']+)\'\s*,\s*\'([^\']+)\'\s*\);/m', $contents, $matches, PREG_SET_ORDER);

        // For each matched apiResource, generate the corresponding resource routes.
        foreach ($matches as $match) {
            $resource = $match[1];
            $controller = $match[2];
            $output = array_merge($output, $this->generateApiResourceRoutes($resource, $controller));
        }

        // This regex matches individual route declarations (e.g., Route::get(),
        // Route::post(), etc.). It captures the HTTP method, the URI, and the
        // controller and method names.
        preg_match_all('/Route::(get|post|put|patch|delete)\(\s*\'([^\']+)\'\s*,\s*\[\s*\'([^\']+)\'\s*,\s*\'([^\']+)\'\s*\]\s*\);/m', $contents, $matches, PREG_SET_ORDER);

        // For each matched route, create a route object and add it to the output.
        foreach ($matches as $match) {
            $method = strtoupper($match[1]);
            $uri = 'api/' . $match[2];
            $actionName = $match[4];

            $output[] = [
                'method' => $method,
                'uri' => $uri,
                'name' => null,
                'action' => $match[3] . '@' . $actionName,
                'action_name' => $actionName,
                'parameters' => $this->extractParameters($uri),
            ];
        }

        // Output the parsed routes as a JSON string.
        echo json_encode($output, JSON_PRETTY_PRINT);
    }

    /**
     * Generates the routes for an API resource.
     *
     * @param string $resource The name of the resource (e.g., "users").
     * @param string $controller The name of the controller (e.g., "UserController").
     * @return array An array of resource routes.
     */
    private function generateApiResourceRoutes($resource, $controller)
    {
        $routes = [];
        $baseUri = "api/{$resource}";
        // The resource parameter (e.g., "{user}").
        $resourceParam = "{" . rtrim($resource, 's') . "}";

        // The standard API resource actions and their corresponding HTTP methods and URIs.
        $actions = [
            'index' => ['method' => 'GET', 'uri' => $baseUri],
            'store' => ['method' => 'POST', 'uri' => $baseUri],
            'show' => ['method' => 'GET', 'uri' => "{$baseUri}/{$resourceParam}"],
            'update' => ['method' => 'PUT', 'uri' => "{$baseUri}/{$resourceParam}"],
            'destroy' => ['method' => 'DELETE', 'uri' => "{$baseUri}/{$resourceParam}"],
        ];

        // For each action, create a route object and add it to the routes array.
        foreach ($actions as $action => $details) {
            $routes[] = [
                'method' => $details['method'],
                'uri' => $details['uri'],
                'name' => "{$resource}.{$action}",
                'action' => "{$controller}@{$action}",
                'action_name' => $this->actionMap[$action] ?? $action,
                'parameters' => $this->extractParameters($details['uri']),
            ];
        }

        return $routes;
    }

    /**
     * Extracts the parameters from a URI.
     *
     * @param string $uri The URI to extract the parameters from.
     * @return array An array of parameters.
     */
    private function extractParameters($uri)
    {
        // This regex matches any string enclosed in curly braces (e.g., "{user}").
        preg_match_all('/\{([^\}]+)\}/', $uri, $matches);
        return $matches[1];
    }
}

// Create a new ApiGenerator and run it.
(new ApiGenerator())->generate();
