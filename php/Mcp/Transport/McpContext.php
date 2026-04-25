<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Front\Mcp;

use Closure;
use Illuminate\Http\Request;

final class McpContext
{
    public function __construct(
        private ?string $sessionId = null,
        private ?object $currentPlan = null,
        private ?Closure $notificationCallback = null,
        private ?Closure $logCallback = null,
        private array|object|null $scopeSource = null,
    ) {}

    public function getSessionId(): ?string
    {
        return $this->sessionId;
    }

    public function setSessionId(?string $sessionId): void
    {
        $this->sessionId = $sessionId;
    }

    public function getCurrentPlan(): ?object
    {
        return $this->currentPlan;
    }

    public function setCurrentPlan(?object $plan): void
    {
        $this->currentPlan = $plan;
    }

    public function sendNotification(string $method, array $params = []): void
    {
        if ($this->notificationCallback instanceof Closure) {
            ($this->notificationCallback)($method, $params);
        }
    }

    public function logToSession(string $message, string $type = 'info', array $data = []): void
    {
        if ($this->logCallback instanceof Closure) {
            ($this->logCallback)($message, $type, $data);
        }
    }

    public function setNotificationCallback(?Closure $callback): void
    {
        $this->notificationCallback = $callback;
    }

    public function setLogCallback(?Closure $callback): void
    {
        $this->logCallback = $callback;
    }

    /**
     * @return array<int, string>
     */
    public function getScopes(): array
    {
        foreach ($this->scopeCandidates() as $candidate) {
            $scopes = $this->resolveScopes($candidate);

            if ($scopes !== []) {
                return $scopes;
            }
        }

        return [];
    }

    public function hasScope(string $scope): bool
    {
        $wanted = $this->canonicalScope($scope);
        if ($wanted === null) {
            return false;
        }

        foreach ($this->getScopes() as $grantedScope) {
            if ($this->canonicalScope($grantedScope) === $wanted) {
                return true;
            }
        }

        return false;
    }

    public function hasSession(): bool
    {
        return $this->sessionId !== null;
    }

    public function hasPlan(): bool
    {
        return $this->currentPlan !== null;
    }

    /**
     * @return array<int, array|object>
     */
    private function scopeCandidates(): array
    {
        $candidates = [];

        if (is_array($this->scopeSource) || is_object($this->scopeSource)) {
            $candidates[] = $this->scopeSource;
        }

        if ($this->currentPlan !== null) {
            $candidates[] = $this->currentPlan;
        }

        $request = $this->currentRequest();
        if (! $request instanceof Request) {
            return $candidates;
        }

        $requestContext = $request->attributes->get('mcp_workspace_context');
        if (is_array($requestContext) || is_object($requestContext)) {
            $candidates[] = $requestContext;
        }

        foreach (['agent_api_key', 'api_key'] as $attribute) {
            $value = $request->attributes->get($attribute);

            if (is_array($value) || is_object($value)) {
                $candidates[] = $value;
            }
        }

        return $candidates;
    }

    /**
     * @return array<int, string>
     */
    private function resolveScopes(mixed $source, int $depth = 0): array
    {
        if ($depth > 3 || $source === null || $source === $this) {
            return [];
        }

        if (is_string($source)) {
            return $this->normaliseScopes([$source]);
        }

        if (is_array($source)) {
            if (array_is_list($source)) {
                return $this->normaliseScopes($source);
            }

            foreach (['scopes', 'permissions', 'authorised_scopes', 'authorized_scopes'] as $key) {
                if (! array_key_exists($key, $source)) {
                    continue;
                }

                $scopes = $this->resolveScopes($source[$key], $depth + 1);
                if ($scopes !== []) {
                    return $scopes;
                }
            }

            foreach (['api_key', 'agent_api_key', 'apiKey', 'agentApiKey', 'session', 'auth', 'authorisation', 'authorization'] as $key) {
                if (! array_key_exists($key, $source)) {
                    continue;
                }

                $scopes = $this->resolveScopes($source[$key], $depth + 1);
                if ($scopes !== []) {
                    return $scopes;
                }
            }

            return [];
        }

        if (! is_object($source)) {
            return [];
        }

        foreach (['getScopes', 'getPermissions'] as $method) {
            if (! method_exists($source, $method)) {
                continue;
            }

            $scopes = $this->resolveScopes($source->{$method}(), $depth + 1);
            if ($scopes !== []) {
                return $scopes;
            }
        }

        foreach (['scopes', 'permissions', 'authorised_scopes', 'authorized_scopes'] as $key) {
            $scopes = $this->resolveScopes($this->extractObjectValue($source, $key), $depth + 1);
            if ($scopes !== []) {
                return $scopes;
            }
        }

        foreach (['apiKey', 'agentApiKey', 'api_key', 'agent_api_key', 'session', 'auth', 'authorisation', 'authorization'] as $key) {
            $scopes = $this->resolveScopes($this->extractObjectValue($source, $key), $depth + 1);
            if ($scopes !== []) {
                return $scopes;
            }
        }

        return [];
    }

    private function extractObjectValue(object $source, string $key): mixed
    {
        if (method_exists($source, 'getAttribute')) {
            $value = $source->getAttribute($key);

            if ($value !== null) {
                return $value;
            }
        }

        foreach ($this->getterNames($key) as $getter) {
            if (! method_exists($source, $getter)) {
                continue;
            }

            $value = $source->{$getter}();
            if ($value !== null) {
                return $value;
            }
        }

        if (property_exists($source, $key)) {
            return $source->{$key};
        }

        $vars = get_object_vars($source);
        if (array_key_exists($key, $vars)) {
            return $vars[$key];
        }

        if (isset($source->{$key})) {
            return $source->{$key};
        }

        return null;
    }

    /**
     * @return array<int, string>
     */
    private function getterNames(string $key): array
    {
        $segments = array_filter(explode('_', $key), fn (string $segment): bool => $segment !== '');
        $studly = implode('', array_map(static fn (string $segment): string => ucfirst($segment), $segments));

        $names = ['get'.ucfirst($key)];
        if ($studly !== '') {
            $names[] = 'get'.$studly;
        }

        return array_values(array_unique($names));
    }

    private function currentRequest(): ?Request
    {
        if (! function_exists('app') || ! class_exists(Request::class) || ! app()->bound('request')) {
            return null;
        }

        $request = app('request');

        return $request instanceof Request ? $request : null;
    }

    /**
     * @param  array<int, mixed>  $scopes
     * @return array<int, string>
     */
    private function normaliseScopes(array $scopes): array
    {
        $normalised = [];
        $seen = [];

        foreach ($scopes as $scope) {
            if (! is_string($scope)) {
                continue;
            }

            $cleanScope = trim($scope);
            $canonicalScope = $this->canonicalScope($cleanScope);

            if ($canonicalScope === null || isset($seen[$canonicalScope])) {
                continue;
            }

            $seen[$canonicalScope] = true;
            $normalised[] = $cleanScope;
        }

        return $normalised;
    }

    private function canonicalScope(string $scope): ?string
    {
        $cleanScope = trim($scope);

        if ($cleanScope === '') {
            return null;
        }

        return str_replace(':', '.', $cleanScope);
    }
}
