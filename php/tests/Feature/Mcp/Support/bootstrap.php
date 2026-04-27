<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

function mcpRequire(string $relativePath): void
{
    require_once dirname(__DIR__, 4).'/'.$relativePath;
}

function mcpDefineLaravelMcpStubs(): void
{
    if (class_exists('Laravel\\Mcp\\Request') && class_exists('Laravel\\Mcp\\Response') && class_exists('Laravel\\Mcp\\Server\\Resource')) {
        return;
    }

    eval(<<<'PHP'
    namespace Laravel\Mcp {
        class Request
        {
            public function __construct(private array $data = [])
            {
            }

            public function get(string $key, mixed $default = null): mixed
            {
                return $this->data[$key] ?? $default;
            }
        }

        class Response
        {
            public function __construct(
                public string $type,
                public string $content,
            ) {
            }

            public static function text(string $content): self
            {
                return new self('text', $content);
            }
        }
    }

    namespace Laravel\Mcp\Server {
        abstract class Resource
        {
            protected string $description = '';
        }
    }
    PHP);
}

function mcpInvokeProtected(object $object, string $method, array $arguments = []): mixed
{
    $reflection = new ReflectionMethod($object, $method);
    $reflection->setAccessible(true);

    return $reflection->invokeArgs($object, $arguments);
}
