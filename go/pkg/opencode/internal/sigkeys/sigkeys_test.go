// SPDX-License-Identifier: EUPL-1.2

package sigkeys

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"

	core "dappco.re/go"
)

// — Verify ——————————————————————————————————————————————————————————

func TestSigkeys_Verify_Good(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	canonical := []byte("the canonical signing bytes")
	sig := ed25519.Sign(priv, canonical)

	result := Verify(pub, canonical, sig)

	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestSigkeys_Verify_Bad(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	canonical := []byte("the canonical signing bytes")
	sig := ed25519.Sign(priv, canonical)

	_, priv2, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	sigFromOtherKey := ed25519.Sign(priv2, canonical)

	tests := []struct {
		name     string
		pubkey   ed25519.PublicKey
		data     []byte
		sig      []byte
		wantCode string
	}{
		{
			name:     "wrong public key length (empty)",
			pubkey:   ed25519.PublicKey{},
			data:     canonical,
			sig:      sig,
			wantCode: sigCorruptReason,
		},
		{
			name:     "wrong public key length (too short, 31 bytes)",
			pubkey:   ed25519.PublicKey(pub[:31]),
			data:     canonical,
			sig:      sig,
			wantCode: sigCorruptReason,
		},
		{
			name:     "wrong public key length (too long, 33 bytes)",
			pubkey:   ed25519.PublicKey(append(pub, 0)),
			data:     canonical,
			sig:      sig,
			wantCode: sigCorruptReason,
		},
		{
			name:     "wrong signature length (empty)",
			pubkey:   pub,
			data:     canonical,
			sig:      []byte{},
			wantCode: sigCorruptReason,
		},
		{
			name:     "wrong signature length (too short, 10 bytes)",
			pubkey:   pub,
			data:     canonical,
			sig:      sig[:10],
			wantCode: sigCorruptReason,
		},
		{
			name:     "wrong signature length (too long, 65 bytes)",
			pubkey:   pub,
			data:     canonical,
			sig:      append(sig, 0),
			wantCode: sigCorruptReason,
		},
		{
			name:     "wrong message (signature mismatch on different canonical bytes)",
			pubkey:   pub,
			data:     []byte("different canonical bytes"),
			sig:      sig,
			wantCode: sigInvalidReason,
		},
		{
			name:     "wrong key (signature from different keypair)",
			pubkey:   pub,
			data:     canonical,
			sig:      sigFromOtherKey,
			wantCode: sigInvalidReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Verify(tt.pubkey, tt.data, tt.sig)
			core.AssertFalse(t, result.OK)
			core.AssertContains(t, result.Error(), tt.wantCode)
		})
	}
}

func TestSigkeys_Verify_Ugly(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)

	sigForEmpty := ed25519.Sign(priv, nil)
	sigForMsg := ed25519.Sign(priv, []byte("msg"))

	tests := []struct {
		name     string
		pubkey   ed25519.PublicKey
		data     []byte
		sig      []byte
		expectOk bool
		wantCode string
	}{
		{
			name:     "nil public key",
			pubkey:   nil,
			data:     []byte("msg"),
			sig:      sigForMsg,
			expectOk: false,
			wantCode: sigCorruptReason,
		},
		{
			name:     "nil canonical bytes with matching empty-message signature",
			pubkey:   pub,
			data:     nil,
			sig:      sigForEmpty,
			expectOk: true,
		},
		{
			name:     "nil canonical bytes with non-matching signature",
			pubkey:   pub,
			data:     nil,
			sig:      sigForMsg,
			expectOk: false,
			wantCode: sigInvalidReason,
		},
		{
			name:     "zero-value signature (nil) with valid data",
			pubkey:   pub,
			data:     []byte("msg"),
			sig:      nil,
			expectOk: false,
			wantCode: sigCorruptReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Verify(tt.pubkey, tt.data, tt.sig)
			core.AssertEqual(t, tt.expectOk, result.OK)
			if !tt.expectOk {
				core.AssertContains(t, result.Error(), tt.wantCode)
			} else {
				core.AssertNil(t, result.Value)
			}
		})
	}
}

// — ParsePublicKey ——————————————————————————————————————————————————

func TestSigkeys_ParsePublicKey_Good(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	b64 := base64.StdEncoding.EncodeToString(pub)
	paddedB64 := "  " + b64 + "\n"

	tests := []struct {
		name  string
		input string
	}{
		{name: "clean base64 key", input: b64},
		{name: "base64 key with surrounding whitespace", input: paddedB64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePublicKey(tt.input)
			core.AssertTrue(t, result.OK)
			parsed, ok := result.Value.(ed25519.PublicKey)
			core.AssertTrue(t, ok)
			core.AssertEqual(t, pub, parsed)
		})
	}
}

func TestSigkeys_ParsePublicKey_Bad(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{
			name:    "invalid base64 characters",
			input:   "!!!not-valid-base64!!!",
			wantMsg: "not valid base64",
		},
		{
			name:    "decoded zero bytes (empty valid base64)",
			input:   base64.StdEncoding.EncodeToString([]byte{}),
			wantMsg: "length 0",
		},
		{
			name:    "decoded 16 bytes (too short for ed25519 key)",
			input:   base64.StdEncoding.EncodeToString(make([]byte, 16)),
			wantMsg: "length 16",
		},
		{
			name:    "decoded 33 bytes (too long for ed25519 key)",
			input:   base64.StdEncoding.EncodeToString(make([]byte, 33)),
			wantMsg: "length 33",
		},
		{
			name:    "decoded 64 bytes (signature-sized, not key-sized)",
			input:   base64.StdEncoding.EncodeToString(make([]byte, 64)),
			wantMsg: "length 64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePublicKey(tt.input)
			core.AssertFalse(t, result.OK)
			core.AssertContains(t, result.Error(), tt.wantMsg)
		})
	}
}

func TestSigkeys_ParsePublicKey_Ugly(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOk  bool
		wantMsg string
	}{
		{
			name:    "empty string (valid base64 decoding to zero bytes)",
			input:   "",
			wantOk:  false,
			wantMsg: "length 0",
		},
		{
			name:    "whitespace only string",
			input:   "   \t \n  ",
			wantOk:  false,
			wantMsg: "length 0",
		},
		{
			name:    "very long base64 (1024 decoded bytes)",
			input:   base64.StdEncoding.EncodeToString(make([]byte, 1024)),
			wantOk:  false,
			wantMsg: "length 1024",
		},
		{
			name:    "single character (incomplete base64 input)",
			input:   "A",
			wantOk:  false,
			wantMsg: "not valid base64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePublicKey(tt.input)
			core.AssertEqual(t, tt.wantOk, result.OK)
			if !tt.wantOk {
				core.AssertContains(t, result.Error(), tt.wantMsg)
			}
		})
	}
}
