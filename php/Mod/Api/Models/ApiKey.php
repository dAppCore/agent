<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Models;

use Core\Tenant\Models\User;
use Core\Tenant\Models\Workspace;
use DateTimeInterface;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\SoftDeletes;
use Illuminate\Support\Str;

class ApiKey extends Model
{
    use SoftDeletes;

    public const HASH_BCRYPT = 'bcrypt';

    public const SCOPE_READ = 'read';

    public const SCOPE_WRITE = 'write';

    public const SCOPE_DELETE = 'delete';

    public const DEFAULT_SCOPES = [
        self::SCOPE_READ,
        self::SCOPE_WRITE,
    ];

    protected $fillable = [
        'workspace_id',
        'user_id',
        'name',
        'key',
        'hash_algorithm',
        'prefix',
        'scopes',
        'allowed_ips',
        'last_used_at',
        'expires_at',
        'grace_period_ends_at',
    ];

    protected $casts = [
        'scopes' => 'array',
        'allowed_ips' => 'array',
        'last_used_at' => 'datetime',
        'expires_at' => 'datetime',
        'grace_period_ends_at' => 'datetime',
    ];

    protected $hidden = [
        'key',
    ];

    /**
     * Create a bcrypt-backed API key using the hk_xxxxxxxx_xxxxx format.
     *
     * Example:
     * `ApiKey::generate(12, null, 'MCP Gateway')`
     */
    public static function generate(
        int $workspaceId,
        ?int $userId,
        string $name,
        array $scopes = self::DEFAULT_SCOPES,
        ?DateTimeInterface $expiresAt = null
    ): array {
        $key = Str::random(48);
        $prefix = 'hk_'.Str::random(8);

        $apiKey = static::query()->create([
            'workspace_id' => $workspaceId,
            'user_id' => $userId,
            'name' => $name,
            'key' => password_hash($key, PASSWORD_BCRYPT),
            'hash_algorithm' => self::HASH_BCRYPT,
            'prefix' => $prefix,
            'scopes' => array_values($scopes),
            'expires_at' => $expiresAt,
        ]);

        return [
            'api_key' => $apiKey,
            'plain_key' => "{$prefix}_{$key}",
        ];
    }

    /**
     * Find a stored API key using the plain hk_ token format.
     *
     * The lookup is prefix-first, then password_verify() on each candidate.
     * We never hash the plain token and query the hash column directly.
     */
    public static function findByPlainKey(string $plainKey): ?static
    {
        if (! str_starts_with($plainKey, 'hk_')) {
            return null;
        }

        $parts = explode('_', $plainKey, 3);
        if (count($parts) !== 3 || strlen($parts[1]) !== 8 || strlen($parts[2]) !== 48) {
            return null;
        }

        $prefix = $parts[0].'_'.$parts[1];
        $secret = $parts[2];

        $candidates = static::query()
            ->where('prefix', $prefix)
            ->where(function ($query) {
                $query->whereNull('expires_at')
                    ->orWhere('expires_at', '>', now());
            })
            ->where(function ($query) {
                $query->whereNull('grace_period_ends_at')
                    ->orWhere('grace_period_ends_at', '>', now());
            })
            ->get();

        foreach ($candidates as $candidate) {
            if ($candidate->verifyKey($secret)) {
                return $candidate;
            }
        }

        return null;
    }

    public function verifyKey(string $plainKey): bool
    {
        return password_verify($plainKey, $this->key);
    }

    public function recordUsage(): void
    {
        $this->forceFill(['last_used_at' => now()])->save();
    }

    public function revoke(): void
    {
        $this->delete();
    }

    public function isExpired(): bool
    {
        return $this->expires_at !== null && $this->expires_at->isPast();
    }

    public function hasScope(string $scope): bool
    {
        return in_array($scope, $this->scopes ?? [], true);
    }

    public function hasScopes(array $scopes): bool
    {
        foreach ($scopes as $scope) {
            if (! $this->hasScope((string) $scope)) {
                return false;
            }
        }

        return true;
    }

    public function hasIpRestrictions(): bool
    {
        return ($this->allowed_ips ?? []) !== [];
    }

    /**
     * @return array<int, string>
     */
    public function getAllowedIps(): array
    {
        return array_values($this->allowed_ips ?? []);
    }

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class, 'workspace_id');
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class, 'user_id');
    }
}
