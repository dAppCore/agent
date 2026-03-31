<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Str;

/**
 * Agent API Key - enables external agent access to Host Hub.
 *
 * Keys are hashed for storage. The plaintext key is only
 * available once during creation.
 *
 * @property int $id
 * @property int $workspace_id
 * @property string $name
 * @property string $key
 * @property array $permissions
 * @property int $rate_limit
 * @property int $call_count
 * @property \Carbon\Carbon|null $last_used_at
 * @property \Carbon\Carbon|null $expires_at
 * @property \Carbon\Carbon|null $revoked_at
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 * @property bool $ip_restriction_enabled
 * @property array|null $ip_whitelist
 * @property string|null $last_used_ip
 */
class AgentApiKey extends Model
{
    protected $fillable = [
        'workspace_id',
        'name',
        'key',
        'permissions',
        'rate_limit',
        'call_count',
        'last_used_at',
        'expires_at',
        'revoked_at',
        'ip_restriction_enabled',
        'ip_whitelist',
        'last_used_ip',
    ];

    protected $casts = [
        'permissions' => 'array',
        'rate_limit' => 'integer',
        'call_count' => 'integer',
        'last_used_at' => 'datetime',
        'expires_at' => 'datetime',
        'revoked_at' => 'datetime',
        'ip_restriction_enabled' => 'boolean',
        'ip_whitelist' => 'array',
    ];

    protected $hidden = [
        'key',
    ];

    /**
     * The plaintext key (only available after creation).
     */
    public ?string $plainTextKey = null;

    // Permission constants
    public const PERM_PLANS_READ = 'plans.read';

    public const PERM_PLANS_WRITE = 'plans.write';

    public const PERM_PHASES_WRITE = 'phases.write';

    public const PERM_SESSIONS_READ = 'sessions.read';

    public const PERM_SESSIONS_WRITE = 'sessions.write';

    public const PERM_BRAIN_READ = 'brain.read';

    public const PERM_BRAIN_WRITE = 'brain.write';

    public const PERM_ISSUES_READ = 'issues.read';

    public const PERM_ISSUES_WRITE = 'issues.write';

    public const PERM_SPRINTS_READ = 'sprints.read';

    public const PERM_SPRINTS_WRITE = 'sprints.write';

    public const PERM_MESSAGES_READ = 'messages.read';

    public const PERM_MESSAGES_WRITE = 'messages.write';

    public const PERM_AUTH_WRITE = 'auth.write';

    public const PERM_FLEET_READ = 'fleet.read';

    public const PERM_FLEET_WRITE = 'fleet.write';

    public const PERM_SYNC_READ = 'sync.read';

    public const PERM_SYNC_WRITE = 'sync.write';

    public const PERM_CREDITS_READ = 'credits.read';

    public const PERM_CREDITS_WRITE = 'credits.write';

    public const PERM_SUBSCRIPTION_READ = 'subscription.read';

    public const PERM_SUBSCRIPTION_WRITE = 'subscription.write';

    public const PERM_TOOLS_READ = 'tools.read';

    public const PERM_TEMPLATES_READ = 'templates.read';

    public const PERM_TEMPLATES_INSTANTIATE = 'templates.instantiate';

    // Notify module permissions
    public const PERM_NOTIFY_READ = 'notify:read';

    public const PERM_NOTIFY_WRITE = 'notify:write';

    public const PERM_NOTIFY_SEND = 'notify:send';

    /**
     * All available permissions with descriptions.
     */
    public static function availablePermissions(): array
    {
        return [
            self::PERM_PLANS_READ => 'List and view plans',
            self::PERM_PLANS_WRITE => 'Create, update, archive plans',
            self::PERM_PHASES_WRITE => 'Update phase status, add/complete tasks',
            self::PERM_SESSIONS_READ => 'List and view sessions',
            self::PERM_SESSIONS_WRITE => 'Start, update, complete sessions',
            self::PERM_BRAIN_READ => 'Recall and list brain memories',
            self::PERM_BRAIN_WRITE => 'Store and forget brain memories',
            self::PERM_ISSUES_READ => 'List and view issues',
            self::PERM_ISSUES_WRITE => 'Create, update, and archive issues',
            self::PERM_SPRINTS_READ => 'List and view sprints',
            self::PERM_SPRINTS_WRITE => 'Create, update, and archive sprints',
            self::PERM_MESSAGES_READ => 'Read inbox and conversation threads',
            self::PERM_MESSAGES_WRITE => 'Send and acknowledge messages',
            self::PERM_AUTH_WRITE => 'Provision and revoke agent API keys',
            self::PERM_FLEET_READ => 'View fleet nodes, tasks, and stats',
            self::PERM_FLEET_WRITE => 'Register nodes and manage fleet tasks',
            self::PERM_SYNC_READ => 'Pull shared fleet context and sync status',
            self::PERM_SYNC_WRITE => 'Push dispatch history to the platform',
            self::PERM_CREDITS_READ => 'View agent credit balances and history',
            self::PERM_CREDITS_WRITE => 'Award agent credits',
            self::PERM_SUBSCRIPTION_READ => 'View node budgets and capability detection',
            self::PERM_SUBSCRIPTION_WRITE => 'Update node budgets',
            self::PERM_TOOLS_READ => 'View tool analytics',
            self::PERM_TEMPLATES_READ => 'List and view templates',
            self::PERM_TEMPLATES_INSTANTIATE => 'Create plans from templates',
            // Notify module
            self::PERM_NOTIFY_READ => 'List and view push campaigns, subscribers, and websites',
            self::PERM_NOTIFY_WRITE => 'Create, update, and delete campaigns and subscribers',
            self::PERM_NOTIFY_SEND => 'Send push notifications immediately or schedule sends',
        ];
    }

    // Relationships
    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    // Scopes
    public function scopeActive(Builder $query): Builder
    {
        return $query->whereNull('revoked_at')
            ->where(function ($q) {
                $q->whereNull('expires_at')
                    ->orWhere('expires_at', '>', now());
            });
    }

    public function scopeForWorkspace($query, Workspace|int $workspace)
    {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;

        return $query->where('workspace_id', $workspaceId);
    }

    public function scopeRevoked(Builder $query): Builder
    {
        return $query->whereNotNull('revoked_at');
    }

    public function scopeExpired(Builder $query): Builder
    {
        return $query->whereNotNull('expires_at')
            ->where('expires_at', '<=', now());
    }

    // Factory
    public static function generate(
        Workspace|int $workspace,
        string $name,
        array $permissions = [],
        int $rateLimit = 100,
        ?\Carbon\Carbon $expiresAt = null
    ): self {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;

        // Generate a random key with prefix for identification
        $plainKey = 'ak_'.Str::random(32);

        // Hash using Argon2id for secure storage
        // This provides protection against rainbow table attacks and brute force
        $hashedKey = password_hash($plainKey, PASSWORD_ARGON2ID, [
            'memory_cost' => PASSWORD_ARGON2_DEFAULT_MEMORY_COST,
            'time_cost' => PASSWORD_ARGON2_DEFAULT_TIME_COST,
            'threads' => PASSWORD_ARGON2_DEFAULT_THREADS,
        ]);

        $key = static::create([
            'workspace_id' => $workspaceId,
            'name' => $name,
            'key' => $hashedKey,
            'permissions' => $permissions,
            'rate_limit' => $rateLimit,
            'call_count' => 0,
            'expires_at' => $expiresAt,
        ]);

        // Store plaintext key for one-time display
        $key->plainTextKey = $plainKey;

        return $key;
    }

    /**
     * Find a key by its plaintext value.
     *
     * Note: This requires iterating through all active keys since Argon2id
     * produces unique hashes with embedded salts. Keys are filtered by prefix
     * first for efficiency.
     */
    public static function findByKey(string $plainKey): ?self
    {
        // Early return for obviously invalid keys
        if (! str_starts_with($plainKey, 'ak_') || strlen($plainKey) < 10) {
            return null;
        }

        // Get all active keys and verify against each
        // This is necessary because Argon2id uses unique salts per hash
        $keys = static::whereNull('revoked_at')
            ->where(function ($query) {
                $query->whereNull('expires_at')
                    ->orWhere('expires_at', '>', now());
            })
            ->get();

        foreach ($keys as $key) {
            if (password_verify($plainKey, $key->key)) {
                return $key;
            }
        }

        return null;
    }

    /**
     * Verify if a plaintext key matches this key's hash.
     */
    public function verifyKey(string $plainKey): bool
    {
        return password_verify($plainKey, $this->key);
    }

    // Status helpers
    public function isActive(): bool
    {
        if ($this->revoked_at !== null) {
            return false;
        }

        if ($this->expires_at !== null && $this->expires_at->isPast()) {
            return false;
        }

        return true;
    }

    public function isRevoked(): bool
    {
        return $this->revoked_at !== null;
    }

    public function isExpired(): bool
    {
        return $this->expires_at !== null && $this->expires_at->isPast();
    }

    // Permission helpers
    public function hasPermission(string $permission): bool
    {
        $wanted = $this->normalisePermission($permission);

        foreach ($this->permissions ?? [] as $granted) {
            if ($this->normalisePermission((string) $granted) === $wanted) {
                return true;
            }
        }

        return false;
    }

    public function hasAnyPermission(array $permissions): bool
    {
        foreach ($permissions as $permission) {
            if ($this->hasPermission($permission)) {
                return true;
            }
        }

        return false;
    }

    public function hasAllPermissions(array $permissions): bool
    {
        foreach ($permissions as $permission) {
            if (! $this->hasPermission($permission)) {
                return false;
            }
        }

        return true;
    }

    protected function normalisePermission(string $permission): string
    {
        return str_replace(':', '.', trim($permission));
    }

    // Actions
    public function revoke(): self
    {
        $this->update(['revoked_at' => now()]);

        return $this;
    }

    public function recordUsage(): self
    {
        $this->increment('call_count');
        $this->update(['last_used_at' => now()]);

        return $this;
    }

    public function updatePermissions(array $permissions): self
    {
        $this->update(['permissions' => $permissions]);

        return $this;
    }

    public function updateRateLimit(int $rateLimit): self
    {
        $this->update(['rate_limit' => $rateLimit]);

        return $this;
    }

    public function extendExpiry(\Carbon\Carbon $expiresAt): self
    {
        $this->update(['expires_at' => $expiresAt]);

        return $this;
    }

    public function removeExpiry(): self
    {
        $this->update(['expires_at' => null]);

        return $this;
    }

    // IP Restriction helpers

    /**
     * Enable IP restrictions for this key.
     */
    public function enableIpRestriction(): self
    {
        $this->update(['ip_restriction_enabled' => true]);

        return $this;
    }

    /**
     * Disable IP restrictions for this key.
     */
    public function disableIpRestriction(): self
    {
        $this->update(['ip_restriction_enabled' => false]);

        return $this;
    }

    /**
     * Update the IP whitelist.
     *
     * @param  array<string>  $whitelist
     */
    public function updateIpWhitelist(array $whitelist): self
    {
        $this->update(['ip_whitelist' => $whitelist]);

        return $this;
    }

    /**
     * Add an IP or CIDR to the whitelist.
     */
    public function addToIpWhitelist(string $ipOrCidr): self
    {
        $whitelist = $this->ip_whitelist ?? [];

        if (! in_array($ipOrCidr, $whitelist, true)) {
            $whitelist[] = $ipOrCidr;
            $this->update(['ip_whitelist' => $whitelist]);
        }

        return $this;
    }

    /**
     * Remove an IP or CIDR from the whitelist.
     */
    public function removeFromIpWhitelist(string $ipOrCidr): self
    {
        $whitelist = $this->ip_whitelist ?? [];
        $whitelist = array_values(array_filter($whitelist, fn ($entry) => $entry !== $ipOrCidr));
        $this->update(['ip_whitelist' => $whitelist]);

        return $this;
    }

    /**
     * Record the last used IP address.
     */
    public function recordLastUsedIp(string $ip): self
    {
        $this->update(['last_used_ip' => $ip]);

        return $this;
    }

    /**
     * Check if IP restrictions are enabled and configured.
     */
    public function hasIpRestrictions(): bool
    {
        return $this->ip_restriction_enabled && ! empty($this->ip_whitelist);
    }

    /**
     * Get the count of whitelisted entries.
     */
    public function getIpWhitelistCount(): int
    {
        return count($this->ip_whitelist ?? []);
    }

    // Rate limiting
    public function isRateLimited(): bool
    {
        // Check calls in the last minute
        $recentCalls = $this->getRecentCallCount();

        return $recentCalls >= $this->rate_limit;
    }

    public function getRecentCallCount(int $seconds = 60): int
    {
        // Use Laravel's cache to track calls per minute
        // The AgentApiKeyService increments this key on each authenticated request
        $cacheKey = "agent_api_key_rate:{$this->id}";

        return (int) \Illuminate\Support\Facades\Cache::get($cacheKey, 0);
    }

    public function getRemainingCalls(): int
    {
        return max(0, $this->rate_limit - $this->getRecentCallCount());
    }

    // Display helpers
    public function getMaskedKey(): string
    {
        // Show first 6 chars of the hashed key (not the plaintext)
        return 'ak_'.substr($this->key, 0, 6).'...';
    }

    public function getStatusLabel(): string
    {
        if ($this->isRevoked()) {
            return 'Revoked';
        }

        if ($this->isExpired()) {
            return 'Expired';
        }

        return 'Active';
    }

    public function getStatusColor(): string
    {
        if ($this->isRevoked()) {
            return 'red';
        }

        if ($this->isExpired()) {
            return 'amber';
        }

        return 'green';
    }

    public function getLastUsedForHumans(): string
    {
        if (! $this->last_used_at) {
            return 'Never';
        }

        return $this->last_used_at->diffForHumans();
    }

    public function getExpiresForHumans(): string
    {
        if (! $this->expires_at) {
            return 'Never';
        }

        if ($this->isExpired()) {
            return 'Expired '.$this->expires_at->diffForHumans();
        }

        return 'Expires '.$this->expires_at->diffForHumans();
    }

    // Output
    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'workspace_id' => $this->workspace_id,
            'name' => $this->name,
            'permissions' => $this->permissions,
            'rate_limit' => $this->rate_limit,
            'call_count' => $this->call_count,
            'last_used_at' => $this->last_used_at?->toIso8601String(),
            'expires_at' => $this->expires_at?->toIso8601String(),
            'revoked_at' => $this->revoked_at?->toIso8601String(),
            'status' => $this->getStatusLabel(),
            'ip_restriction_enabled' => $this->ip_restriction_enabled,
            'ip_whitelist_count' => $this->getIpWhitelistCount(),
            'last_used_ip' => $this->last_used_ip,
            'created_at' => $this->created_at?->toIso8601String(),
        ];
    }
}
