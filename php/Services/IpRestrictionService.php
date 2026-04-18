<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\AgentApiKey;

/**
 * IP Restriction Service.
 *
 * Validates IP addresses against API key whitelists.
 * Supports individual IPs and CIDR notation for both IPv4 and IPv6.
 */
class IpRestrictionService
{
    /**
     * Validate if an IP is allowed for the given API key.
     *
     * Returns true if:
     * - IP restrictions are disabled for the key
     * - IP is in the whitelist (exact match or CIDR match)
     */
    public function validateIp(AgentApiKey $apiKey, string $requestIp): bool
    {
        // If IP restrictions are disabled, allow all
        if (! $apiKey->ip_restriction_enabled) {
            return true;
        }

        $whitelist = $apiKey->ip_whitelist ?? [];

        // Empty whitelist with restrictions enabled = deny all
        if (empty($whitelist)) {
            return false;
        }

        return $this->isIpInWhitelist($requestIp, $whitelist);
    }

    /**
     * Check if an IP address is in a whitelist.
     *
     * Supports:
     * - Individual IPv4 addresses (192.168.1.1)
     * - Individual IPv6 addresses (::1, 2001:db8::1)
     * - CIDR notation for IPv4 (192.168.1.0/24)
     * - CIDR notation for IPv6 (2001:db8::/32)
     *
     * @param  array<string>  $whitelist
     */
    public function isIpInWhitelist(string $ip, array $whitelist): bool
    {
        $ip = trim($ip);

        // Validate the request IP is a valid IP address
        if (! filter_var($ip, FILTER_VALIDATE_IP)) {
            return false;
        }

        foreach ($whitelist as $entry) {
            $entry = trim($entry);

            if (empty($entry)) {
                continue;
            }

            // Check for CIDR notation
            if (str_contains($entry, '/')) {
                if ($this->ipMatchesCidr($ip, $entry)) {
                    return true;
                }
            } else {
                // Exact IP match (normalise both for comparison)
                if ($this->normaliseIp($ip) === $this->normaliseIp($entry)) {
                    return true;
                }
            }
        }

        return false;
    }

    /**
     * Check if an IP matches a CIDR range.
     */
    public function ipMatchesCidr(string $ip, string $cidr): bool
    {
        $parts = explode('/', $cidr, 2);

        if (count($parts) !== 2) {
            return false;
        }

        [$range, $prefix] = $parts;
        $prefix = (int) $prefix;

        // Validate both IPs
        if (! filter_var($ip, FILTER_VALIDATE_IP)) {
            return false;
        }

        if (! filter_var($range, FILTER_VALIDATE_IP)) {
            return false;
        }

        $isIpv6 = filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV6);
        $isRangeIpv6 = filter_var($range, FILTER_VALIDATE_IP, FILTER_FLAG_IPV6);

        // IP version must match
        if ($isIpv6 !== $isRangeIpv6) {
            return false;
        }

        if ($isIpv6) {
            return $this->ipv6MatchesCidr($ip, $range, $prefix);
        }

        return $this->ipv4MatchesCidr($ip, $range, $prefix);
    }

    /**
     * Check if an IPv4 address matches a CIDR range.
     */
    protected function ipv4MatchesCidr(string $ip, string $range, int $prefix): bool
    {
        // Validate prefix length
        if ($prefix < 0 || $prefix > 32) {
            return false;
        }

        $ipLong = ip2long($ip);
        $rangeLong = ip2long($range);

        if ($ipLong === false || $rangeLong === false) {
            return false;
        }

        // Create the subnet mask
        $mask = -1 << (32 - $prefix);

        // Apply mask and compare
        return ($ipLong & $mask) === ($rangeLong & $mask);
    }

    /**
     * Check if an IPv6 address matches a CIDR range.
     */
    protected function ipv6MatchesCidr(string $ip, string $range, int $prefix): bool
    {
        // Validate prefix length
        if ($prefix < 0 || $prefix > 128) {
            return false;
        }

        // Convert to binary representation
        $ipBin = $this->ipv6ToBinary($ip);
        $rangeBin = $this->ipv6ToBinary($range);

        if ($ipBin === null || $rangeBin === null) {
            return false;
        }

        // Compare the first $prefix bits
        $prefixBytes = (int) floor($prefix / 8);
        $remainingBits = $prefix % 8;

        // Compare full bytes
        if (substr($ipBin, 0, $prefixBytes) !== substr($rangeBin, 0, $prefixBytes)) {
            return false;
        }

        // Compare remaining bits if any
        if ($remainingBits > 0) {
            $mask = 0xFF << (8 - $remainingBits);
            $ipByte = ord($ipBin[$prefixBytes]);
            $rangeByte = ord($rangeBin[$prefixBytes]);

            if (($ipByte & $mask) !== ($rangeByte & $mask)) {
                return false;
            }
        }

        return true;
    }

    /**
     * Convert an IPv6 address to its binary representation.
     */
    protected function ipv6ToBinary(string $ip): ?string
    {
        $packed = inet_pton($ip);

        if ($packed === false) {
            return null;
        }

        return $packed;
    }

    /**
     * Normalise an IP address for comparison.
     *
     * - IPv4: No change needed
     * - IPv6: Expand to full form for consistent comparison
     */
    public function normaliseIp(string $ip): string
    {
        $ip = trim($ip);

        // Try to pack and unpack for normalisation
        $packed = inet_pton($ip);

        if ($packed === false) {
            return $ip; // Return original if invalid
        }

        // inet_ntop will return normalised form
        $normalised = inet_ntop($packed);

        return $normalised !== false ? $normalised : $ip;
    }

    /**
     * Validate an IP address or CIDR notation.
     *
     * @return array{valid: bool, error: ?string}
     */
    public function validateEntry(string $entry): array
    {
        $entry = trim($entry);

        if (empty($entry)) {
            return ['valid' => false, 'error' => 'Empty entry'];
        }

        // Check for CIDR notation
        if (str_contains($entry, '/')) {
            return $this->validateCidr($entry);
        }

        // Validate as plain IP
        if (! filter_var($entry, FILTER_VALIDATE_IP)) {
            return ['valid' => false, 'error' => 'Invalid IP address'];
        }

        return ['valid' => true, 'error' => null];
    }

    /**
     * Validate CIDR notation.
     *
     * @return array{valid: bool, error: ?string}
     */
    public function validateCidr(string $cidr): array
    {
        $parts = explode('/', $cidr, 2);

        if (count($parts) !== 2) {
            return ['valid' => false, 'error' => 'Invalid CIDR notation'];
        }

        [$ip, $prefix] = $parts;

        // Validate IP portion
        if (! filter_var($ip, FILTER_VALIDATE_IP)) {
            return ['valid' => false, 'error' => 'Invalid IP address in CIDR'];
        }

        // Validate prefix is numeric
        if (! is_numeric($prefix)) {
            return ['valid' => false, 'error' => 'Invalid prefix length'];
        }

        $prefix = (int) $prefix;
        $isIpv6 = filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV6);

        // Validate prefix range
        if ($isIpv6) {
            if ($prefix < 0 || $prefix > 128) {
                return ['valid' => false, 'error' => 'IPv6 prefix must be between 0 and 128'];
            }
        } else {
            if ($prefix < 0 || $prefix > 32) {
                return ['valid' => false, 'error' => 'IPv4 prefix must be between 0 and 32'];
            }
        }

        return ['valid' => true, 'error' => null];
    }

    /**
     * Parse a multi-line string of IPs/CIDRs into an array.
     *
     * @return array{entries: array<string>, errors: array<string>}
     */
    public function parseWhitelistInput(string $input): array
    {
        $lines = preg_split('/[\r\n,]+/', $input);
        $entries = [];
        $errors = [];

        foreach ($lines as $line) {
            $line = trim($line);

            if (empty($line)) {
                continue;
            }

            // Skip comments
            if (str_starts_with($line, '#')) {
                continue;
            }

            $validation = $this->validateEntry($line);

            if ($validation['valid']) {
                $entries[] = $line;
            } else {
                $errors[] = "{$line}: {$validation['error']}";
            }
        }

        return [
            'entries' => $entries,
            'errors' => $errors,
        ];
    }

    /**
     * Format a whitelist array as a multi-line string.
     *
     * @param  array<string>  $whitelist
     */
    public function formatWhitelistForDisplay(array $whitelist): string
    {
        return implode("\n", $whitelist);
    }

    /**
     * Get a human-readable description of a CIDR range.
     */
    public function describeCidr(string $cidr): string
    {
        $parts = explode('/', $cidr, 2);

        if (count($parts) !== 2) {
            return $cidr;
        }

        [$ip, $prefix] = $parts;
        $prefix = (int) $prefix;

        $isIpv6 = filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV6);

        if ($isIpv6) {
            $totalHosts = bcpow('2', (string) (128 - $prefix));

            return "{$cidr} ({$totalHosts} addresses)";
        }

        $totalHosts = 2 ** (32 - $prefix);

        return "{$cidr} ({$totalHosts} addresses)";
    }
}
