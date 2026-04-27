<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Admin\Search;

use Illuminate\Contracts\Support\Arrayable;
use JsonSerializable;

final class SearchResult implements Arrayable, JsonSerializable
{
    /**
     * @param  array<string, mixed>  $metadata
     */
    public function __construct(
        public readonly string $id,
        public readonly string $type,
        public readonly string $title,
        public readonly string $description,
        public readonly string $url,
        public readonly string $icon,
        public readonly array $metadata = [],
    ) {}

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            id: (string) ($data['id'] ?? uniqid('search_', true)),
            type: (string) ($data['type'] ?? 'unknown'),
            title: (string) ($data['title'] ?? ''),
            description: (string) ($data['description'] ?? $data['subtitle'] ?? ''),
            url: (string) ($data['url'] ?? '#'),
            icon: (string) ($data['icon'] ?? 'document'),
            metadata: (array) ($data['metadata'] ?? $data['meta'] ?? []),
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'type' => $this->type,
            'title' => $this->title,
            'description' => $this->description,
            'subtitle' => $this->description,
            'url' => $this->url,
            'icon' => $this->icon,
            'metadata' => $this->metadata,
            'meta' => $this->metadata,
        ];
    }

    /**
     * @return array<string, mixed>
     */
    public function jsonSerialize(): array
    {
        return $this->toArray();
    }
}
