<?php

declare(strict_types=1);

/**
 * Tests for the IpRestrictionService.
 *
 * Covers IPv4/IPv6 validation, CIDR matching, and edge cases.
 */

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\IpRestrictionService;
use Core\Tenant\Models\Workspace;

beforeEach(function (): void {
    $this->workspace = Workspace::factory()->create();
    $this->service = app(IpRestrictionService::class);
});

// =============================================================================
// IPv4 Basic Tests
// =============================================================================

test('validates exact IPv4 match', function (): void {
    $result = $this->service->isIpInWhitelist('192.168.1.100', ['192.168.1.100']);

    expect($result)->toBeTrue();
});

test('rejects non-matching IPv4', function (): void {
    $result = $this->service->isIpInWhitelist('192.168.1.100', ['192.168.1.200']);

    expect($result)->toBeFalse();
});

test('validates IPv4 in multiple entries', function (): void {
    $whitelist = ['10.0.0.1', '192.168.1.100', '172.16.0.1'];

    $result = $this->service->isIpInWhitelist('192.168.1.100', $whitelist);

    expect($result)->toBeTrue();
});

test('rejects invalid IPv4', function (): void {
    $result = $this->service->isIpInWhitelist('invalid', ['192.168.1.100']);

    expect($result)->toBeFalse();
});

test('rejects IPv4 out of range', function (): void {
    $result = $this->service->isIpInWhitelist('256.256.256.256', ['192.168.1.100']);

    expect($result)->toBeFalse();
});

// =============================================================================
// IPv4 CIDR Tests
// =============================================================================

test('validates IPv4 in CIDR /24', function (): void {
    expect($this->service->isIpInWhitelist('192.168.1.0', ['192.168.1.0/24']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.1', ['192.168.1.0/24']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.128', ['192.168.1.0/24']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.255', ['192.168.1.0/24']))->toBeTrue();
});

test('rejects IPv4 outside CIDR /24', function (): void {
    expect($this->service->isIpInWhitelist('192.168.2.0', ['192.168.1.0/24']))->toBeFalse();
    expect($this->service->isIpInWhitelist('192.168.0.255', ['192.168.1.0/24']))->toBeFalse();
});

test('validates IPv4 in CIDR /16', function (): void {
    expect($this->service->isIpInWhitelist('192.168.0.1', ['192.168.0.0/16']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.255.255', ['192.168.0.0/16']))->toBeTrue();
});

test('rejects IPv4 outside CIDR /16', function (): void {
    expect($this->service->isIpInWhitelist('192.169.0.1', ['192.168.0.0/16']))->toBeFalse();
});

test('validates IPv4 in CIDR /8', function (): void {
    expect($this->service->isIpInWhitelist('10.0.0.1', ['10.0.0.0/8']))->toBeTrue();
    expect($this->service->isIpInWhitelist('10.255.255.255', ['10.0.0.0/8']))->toBeTrue();
});

test('rejects IPv4 outside CIDR /8', function (): void {
    expect($this->service->isIpInWhitelist('11.0.0.1', ['10.0.0.0/8']))->toBeFalse();
});

test('validates IPv4 in CIDR /32', function (): void {
    expect($this->service->isIpInWhitelist('192.168.1.100', ['192.168.1.100/32']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.101', ['192.168.1.100/32']))->toBeFalse();
});

test('validates IPv4 in CIDR /0', function (): void {
    // /0 means all IPv4 addresses
    expect($this->service->isIpInWhitelist('1.2.3.4', ['0.0.0.0/0']))->toBeTrue();
    expect($this->service->isIpInWhitelist('255.255.255.255', ['0.0.0.0/0']))->toBeTrue();
});

test('validates IPv4 in non-standard CIDR', function (): void {
    // /28 gives 16 addresses
    expect($this->service->isIpInWhitelist('192.168.1.0', ['192.168.1.0/28']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.15', ['192.168.1.0/28']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.16', ['192.168.1.0/28']))->toBeFalse();
});

test('validates IPv4 in CIDR /25', function (): void {
    // /25 gives 128 addresses (0-127)
    expect($this->service->isIpInWhitelist('192.168.1.0', ['192.168.1.0/25']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.127', ['192.168.1.0/25']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.128', ['192.168.1.0/25']))->toBeFalse();
});

test('validates IPv4 in CIDR /30', function (): void {
    // /30 gives 4 addresses
    expect($this->service->isIpInWhitelist('192.168.1.0', ['192.168.1.0/30']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.3', ['192.168.1.0/30']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.4', ['192.168.1.0/30']))->toBeFalse();
});

test('validates IPv4 in CIDR /31', function (): void {
    // /31 gives 2 addresses
    expect($this->service->isIpInWhitelist('192.168.1.0', ['192.168.1.0/31']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.1', ['192.168.1.0/31']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.2', ['192.168.1.0/31']))->toBeFalse();
});

// =============================================================================
// IPv6 Basic Tests
// =============================================================================

test('validates exact IPv6 match', function (): void {
    $result = $this->service->isIpInWhitelist('2001:db8::1', ['2001:db8::1']);

    expect($result)->toBeTrue();
});

test('validates localhost IPv6', function (): void {
    $result = $this->service->isIpInWhitelist('::1', ['::1']);

    expect($result)->toBeTrue();
});

test('rejects non-matching IPv6', function (): void {
    $result = $this->service->isIpInWhitelist('2001:db8::1', ['2001:db8::2']);

    expect($result)->toBeFalse();
});

test('normalises IPv6 for comparison', function (): void {
    // These are the same address in different formats
    $result = $this->service->isIpInWhitelist(
        '2001:0db8:0000:0000:0000:0000:0000:0001',
        ['2001:db8::1']
    );

    expect($result)->toBeTrue();
});

test('validates full IPv6 address', function (): void {
    $result = $this->service->isIpInWhitelist(
        '2001:0db8:85a3:0000:0000:8a2e:0370:7334',
        ['2001:db8:85a3::8a2e:370:7334']
    );

    expect($result)->toBeTrue();
});

// =============================================================================
// IPv6 CIDR Tests
// =============================================================================

test('validates IPv6 in CIDR /64', function (): void {
    expect($this->service->isIpInWhitelist(
        '2001:db8:abcd:0012::1',
        ['2001:db8:abcd:0012::/64']
    ))->toBeTrue();
    expect($this->service->isIpInWhitelist(
        '2001:db8:abcd:0012:ffff:ffff:ffff:ffff',
        ['2001:db8:abcd:0012::/64']
    ))->toBeTrue();
});

test('rejects IPv6 outside CIDR /64', function (): void {
    expect($this->service->isIpInWhitelist(
        '2001:db8:abcd:0013::1',
        ['2001:db8:abcd:0012::/64']
    ))->toBeFalse();
});

test('validates IPv6 in CIDR /32', function (): void {
    expect($this->service->isIpInWhitelist(
        '2001:db8:0:0:0:0:0:1',
        ['2001:db8::/32']
    ))->toBeTrue();
    expect($this->service->isIpInWhitelist(
        '2001:db8:ffff:ffff:ffff:ffff:ffff:ffff',
        ['2001:db8::/32']
    ))->toBeTrue();
});

test('rejects IPv6 outside CIDR /32', function (): void {
    expect($this->service->isIpInWhitelist(
        '2001:db9::1',
        ['2001:db8::/32']
    ))->toBeFalse();
});

test('validates IPv6 in CIDR /128', function (): void {
    expect($this->service->isIpInWhitelist(
        '2001:db8::1',
        ['2001:db8::1/128']
    ))->toBeTrue();
    expect($this->service->isIpInWhitelist(
        '2001:db8::2',
        ['2001:db8::1/128']
    ))->toBeFalse();
});

test('validates IPv6 in CIDR /48', function (): void {
    expect($this->service->isIpInWhitelist(
        '2001:db8:abcd::1',
        ['2001:db8:abcd::/48']
    ))->toBeTrue();
    expect($this->service->isIpInWhitelist(
        '2001:db8:abcd:ffff:ffff:ffff:ffff:ffff',
        ['2001:db8:abcd::/48']
    ))->toBeTrue();
    expect($this->service->isIpInWhitelist(
        '2001:db8:abce::1',
        ['2001:db8:abcd::/48']
    ))->toBeFalse();
});

test('validates IPv6 in CIDR /0', function (): void {
    // /0 means all IPv6 addresses
    expect($this->service->isIpInWhitelist('::1', ['::/0']))->toBeTrue();
    expect($this->service->isIpInWhitelist('2001:db8::1', ['::/0']))->toBeTrue();
    expect($this->service->isIpInWhitelist('fe80::1', ['::/0']))->toBeTrue();
});

test('validates IPv6 in CIDR /56', function (): void {
    // /56 is common allocation size
    expect($this->service->isIpInWhitelist(
        '2001:db8:ab00::1',
        ['2001:db8:ab00::/56']
    ))->toBeTrue();
    expect($this->service->isIpInWhitelist(
        '2001:db8:ab00:ff::1',
        ['2001:db8:ab00::/56']
    ))->toBeTrue();
    expect($this->service->isIpInWhitelist(
        '2001:db8:ab01::1',
        ['2001:db8:ab00::/56']
    ))->toBeFalse();
});

// =============================================================================
// IPv4/IPv6 Mixed Tests
// =============================================================================

test('IPv4 does not match IPv6 CIDR', function (): void {
    expect($this->service->isIpInWhitelist(
        '192.168.1.1',
        ['2001:db8::/32']
    ))->toBeFalse();
});

test('IPv6 does not match IPv4 CIDR', function (): void {
    expect($this->service->isIpInWhitelist(
        '2001:db8::1',
        ['192.168.1.0/24']
    ))->toBeFalse();
});

test('whitelist can contain both IPv4 and IPv6', function (): void {
    $whitelist = ['192.168.1.0/24', '2001:db8::/32'];

    expect($this->service->isIpInWhitelist('192.168.1.100', $whitelist))->toBeTrue();
    expect($this->service->isIpInWhitelist('2001:db8::1', $whitelist))->toBeTrue();
    expect($this->service->isIpInWhitelist('10.0.0.1', $whitelist))->toBeFalse();
});

// =============================================================================
// API Key Integration Tests
// =============================================================================

test('validateIp returns true when restrictions disabled', function (): void {
    $key = AgentApiKey::generate($this->workspace, 'Test Key');

    $result = $this->service->validateIp($key, '192.168.1.100');

    expect($result)->toBeTrue();
});

test('validateIp returns false when enabled with empty whitelist', function (): void {
    $key = AgentApiKey::generate($this->workspace, 'Test Key');
    $key->enableIpRestriction();

    $result = $this->service->validateIp($key->fresh(), '192.168.1.100');

    expect($result)->toBeFalse();
});

test('validateIp checks whitelist', function (): void {
    $key = AgentApiKey::generate($this->workspace, 'Test Key');
    $key->enableIpRestriction();
    $key->updateIpWhitelist(['192.168.1.100', '10.0.0.0/8']);

    $fresh = $key->fresh();

    expect($this->service->validateIp($fresh, '192.168.1.100'))->toBeTrue();
    expect($this->service->validateIp($fresh, '10.0.0.50'))->toBeTrue();
    expect($this->service->validateIp($fresh, '172.16.0.1'))->toBeFalse();
});

// =============================================================================
// Entry Validation Tests
// =============================================================================

test('validateEntry accepts valid IPv4', function (): void {
    $result = $this->service->validateEntry('192.168.1.1');

    expect($result['valid'])->toBeTrue();
    expect($result['error'])->toBeNull();
});

test('validateEntry accepts valid IPv6', function (): void {
    $result = $this->service->validateEntry('2001:db8::1');

    expect($result['valid'])->toBeTrue();
    expect($result['error'])->toBeNull();
});

test('validateEntry accepts valid IPv4 CIDR', function (): void {
    $result = $this->service->validateEntry('192.168.1.0/24');

    expect($result['valid'])->toBeTrue();
    expect($result['error'])->toBeNull();
});

test('validateEntry accepts valid IPv6 CIDR', function (): void {
    $result = $this->service->validateEntry('2001:db8::/32');

    expect($result['valid'])->toBeTrue();
    expect($result['error'])->toBeNull();
});

test('validateEntry rejects empty', function (): void {
    $result = $this->service->validateEntry('');

    expect($result['valid'])->toBeFalse();
    expect($result['error'])->toBe('Empty entry');
});

test('validateEntry rejects invalid IP', function (): void {
    $result = $this->service->validateEntry('not-an-ip');

    expect($result['valid'])->toBeFalse();
    expect($result['error'])->toBe('Invalid IP address');
});

test('validateEntry rejects invalid CIDR', function (): void {
    $result = $this->service->validateEntry('192.168.1.0/');

    expect($result['valid'])->toBeFalse();
});

// =============================================================================
// CIDR Validation Tests
// =============================================================================

test('validateCidr accepts valid IPv4 prefixes', function (): void {
    expect($this->service->validateCidr('192.168.1.0/0')['valid'])->toBeTrue();
    expect($this->service->validateCidr('192.168.1.0/16')['valid'])->toBeTrue();
    expect($this->service->validateCidr('192.168.1.0/32')['valid'])->toBeTrue();
});

test('validateCidr rejects invalid IPv4 prefixes', function (): void {
    $result = $this->service->validateCidr('192.168.1.0/33');

    expect($result['valid'])->toBeFalse();
    expect($result['error'])->toContain('IPv4 prefix must be');
});

test('validateCidr accepts valid IPv6 prefixes', function (): void {
    expect($this->service->validateCidr('2001:db8::/0')['valid'])->toBeTrue();
    expect($this->service->validateCidr('2001:db8::/64')['valid'])->toBeTrue();
    expect($this->service->validateCidr('2001:db8::/128')['valid'])->toBeTrue();
});

test('validateCidr rejects invalid IPv6 prefixes', function (): void {
    $result = $this->service->validateCidr('2001:db8::/129');

    expect($result['valid'])->toBeFalse();
    expect($result['error'])->toContain('IPv6 prefix must be');
});

test('validateCidr rejects negative prefix', function (): void {
    $result = $this->service->validateCidr('192.168.1.0/-1');

    expect($result['valid'])->toBeFalse();
});

test('validateCidr rejects non-numeric prefix', function (): void {
    $result = $this->service->validateCidr('192.168.1.0/abc');

    expect($result['valid'])->toBeFalse();
    expect($result['error'])->toBe('Invalid prefix length');
});

test('validateCidr rejects invalid IP in CIDR', function (): void {
    $result = $this->service->validateCidr('invalid/24');

    expect($result['valid'])->toBeFalse();
    expect($result['error'])->toBe('Invalid IP address in CIDR');
});

// =============================================================================
// Parse Whitelist Input Tests
// =============================================================================

test('parseWhitelistInput handles newlines', function (): void {
    $input = "192.168.1.1\n192.168.1.2\n192.168.1.3";

    $result = $this->service->parseWhitelistInput($input);

    expect($result['entries'])->toHaveCount(3);
    expect($result['errors'])->toBeEmpty();
});

test('parseWhitelistInput handles commas', function (): void {
    $input = '192.168.1.1,192.168.1.2,192.168.1.3';

    $result = $this->service->parseWhitelistInput($input);

    expect($result['entries'])->toHaveCount(3);
});

test('parseWhitelistInput handles carriage returns', function (): void {
    $input = "192.168.1.1\r\n192.168.1.2\r\n192.168.1.3";

    $result = $this->service->parseWhitelistInput($input);

    expect($result['entries'])->toHaveCount(3);
});

test('parseWhitelistInput trims whitespace', function (): void {
    $input = "  192.168.1.1  \n  192.168.1.2  ";

    $result = $this->service->parseWhitelistInput($input);

    expect($result['entries'])->toContain('192.168.1.1');
    expect($result['entries'])->toContain('192.168.1.2');
});

test('parseWhitelistInput skips empty lines', function (): void {
    $input = "192.168.1.1\n\n\n192.168.1.2";

    $result = $this->service->parseWhitelistInput($input);

    expect($result['entries'])->toHaveCount(2);
});

test('parseWhitelistInput skips comments', function (): void {
    $input = "# This is a comment\n192.168.1.1\n# Another comment\n192.168.1.2";

    $result = $this->service->parseWhitelistInput($input);

    expect($result['entries'])->toHaveCount(2);
    expect($result['entries'])->not->toContain('# This is a comment');
});

test('parseWhitelistInput collects errors', function (): void {
    $input = "192.168.1.1\ninvalid\n192.168.1.2\nalso-invalid";

    $result = $this->service->parseWhitelistInput($input);

    expect($result['entries'])->toHaveCount(2);
    expect($result['errors'])->toHaveCount(2);
});

test('parseWhitelistInput handles mixed content', function (): void {
    $input = "# Office IPs\n192.168.1.0/24\n# Cloud provider\n10.0.0.0/8\n# Invalid\ninvalid-ip";

    $result = $this->service->parseWhitelistInput($input);

    expect($result['entries'])->toHaveCount(2);
    expect($result['entries'])->toContain('192.168.1.0/24');
    expect($result['entries'])->toContain('10.0.0.0/8');
    expect($result['errors'])->toHaveCount(1);
});

// =============================================================================
// Format Whitelist Tests
// =============================================================================

test('formatWhitelistForDisplay joins with newlines', function (): void {
    $whitelist = ['192.168.1.1', '10.0.0.0/8', '2001:db8::/32'];

    $result = $this->service->formatWhitelistForDisplay($whitelist);

    expect($result)->toBe("192.168.1.1\n10.0.0.0/8\n2001:db8::/32");
});

test('formatWhitelistForDisplay handles empty', function (): void {
    $result = $this->service->formatWhitelistForDisplay([]);

    expect($result)->toBe('');
});

// =============================================================================
// Describe CIDR Tests
// =============================================================================

test('describeCidr for IPv4', function (): void {
    expect($this->service->describeCidr('192.168.1.0/24'))->toContain('256 addresses');
    expect($this->service->describeCidr('192.168.1.0/32'))->toContain('1 addresses');
    expect($this->service->describeCidr('192.168.1.0/0'))->toContain('4294967296 addresses');
});

test('describeCidr for IPv6', function (): void {
    $result = $this->service->describeCidr('2001:db8::/32');

    expect($result)->toContain('2001:db8::/32');
    expect($result)->toContain('addresses');
});

test('describeCidr returns original for invalid', function (): void {
    $result = $this->service->describeCidr('invalid');

    expect($result)->toBe('invalid');
});

// =============================================================================
// Normalise IP Tests
// =============================================================================

test('normaliseIp returns same for IPv4', function (): void {
    $result = $this->service->normaliseIp('192.168.1.1');

    expect($result)->toBe('192.168.1.1');
});

test('normaliseIp compresses IPv6', function (): void {
    $result = $this->service->normaliseIp('2001:0db8:0000:0000:0000:0000:0000:0001');

    expect($result)->toBe('2001:db8::1');
});

test('normaliseIp returns original for invalid', function (): void {
    $result = $this->service->normaliseIp('invalid');

    expect($result)->toBe('invalid');
});

test('normaliseIp handles trimming', function (): void {
    $result = $this->service->normaliseIp('  192.168.1.1  ');

    expect($result)->toBe('192.168.1.1');
});

// =============================================================================
// Edge Cases
// =============================================================================

test('handles trimmed whitelist entries', function (): void {
    $result = $this->service->isIpInWhitelist('192.168.1.1', ['  192.168.1.1  ']);

    expect($result)->toBeTrue();
});

test('skips empty whitelist entries', function (): void {
    $result = $this->service->isIpInWhitelist('192.168.1.1', ['', '192.168.1.1', '']);

    expect($result)->toBeTrue();
});

test('returns false for empty whitelist', function (): void {
    $result = $this->service->isIpInWhitelist('192.168.1.1', []);

    expect($result)->toBeFalse();
});

test('handles loopback addresses', function (): void {
    expect($this->service->isIpInWhitelist('127.0.0.1', ['127.0.0.0/8']))->toBeTrue();
    expect($this->service->isIpInWhitelist('::1', ['::1']))->toBeTrue();
});

test('handles private ranges', function (): void {
    // RFC 1918 private ranges
    expect($this->service->isIpInWhitelist('10.0.0.1', ['10.0.0.0/8']))->toBeTrue();
    expect($this->service->isIpInWhitelist('172.16.0.1', ['172.16.0.0/12']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.0.1', ['192.168.0.0/16']))->toBeTrue();
});

test('handles link-local IPv6', function (): void {
    expect($this->service->isIpInWhitelist('fe80::1', ['fe80::/10']))->toBeTrue();
});

test('handles unique local IPv6', function (): void {
    expect($this->service->isIpInWhitelist('fd00::1', ['fc00::/7']))->toBeTrue();
});

test('rejects malformed CIDR', function (): void {
    expect($this->service->ipMatchesCidr('192.168.1.1', '192.168.1.0'))->toBeFalse();
    expect($this->service->ipMatchesCidr('192.168.1.1', '192.168.1.0//'))->toBeFalse();
});

test('handles multiple CIDR ranges in whitelist', function (): void {
    $whitelist = [
        '10.0.0.0/8',
        '172.16.0.0/12',
        '192.168.0.0/16',
        '2001:db8::/32',
    ];

    expect($this->service->isIpInWhitelist('10.1.2.3', $whitelist))->toBeTrue();
    expect($this->service->isIpInWhitelist('172.20.1.1', $whitelist))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.100.50', $whitelist))->toBeTrue();
    expect($this->service->isIpInWhitelist('2001:db8:1234::1', $whitelist))->toBeTrue();
    expect($this->service->isIpInWhitelist('8.8.8.8', $whitelist))->toBeFalse();
});

test('handles boundary IPs in CIDR range', function (): void {
    // First and last IP in a /24
    expect($this->service->isIpInWhitelist('192.168.1.0', ['192.168.1.0/24']))->toBeTrue();
    expect($this->service->isIpInWhitelist('192.168.1.255', ['192.168.1.0/24']))->toBeTrue();

    // Just outside the range
    expect($this->service->isIpInWhitelist('192.168.0.255', ['192.168.1.0/24']))->toBeFalse();
    expect($this->service->isIpInWhitelist('192.168.2.0', ['192.168.1.0/24']))->toBeFalse();
});

test('handles very large IPv6 ranges', function (): void {
    // /16 gives an enormous number of addresses
    expect($this->service->isIpInWhitelist('2001:db8::1', ['2001::/16']))->toBeTrue();
    expect($this->service->isIpInWhitelist('2001:ffff:ffff:ffff:ffff:ffff:ffff:ffff', ['2001::/16']))->toBeTrue();
    expect($this->service->isIpInWhitelist('2002::1', ['2001::/16']))->toBeFalse();
});
