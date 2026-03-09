<?php

declare(strict_types=1);

namespace Tests\Fixtures;

use Illuminate\Foundation\Auth\User as Authenticatable;

/**
 * Fake user fixture for Livewire component tests.
 *
 * Satisfies the isHades() check used by all admin components.
 */
class HadesUser extends Authenticatable
{
    protected $fillable = ['id', 'name', 'email'];

    protected $table = 'users';

    public $timestamps = false;

    public function isHades(): bool
    {
        return true;
    }

    public function defaultHostWorkspace(): ?object
    {
        return null;
    }

    public function getAuthIdentifier(): mixed
    {
        return $this->attributes['id'] ?? 1;
    }
}
