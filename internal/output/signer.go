package output

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/vettcode/scanner/pkg/models"
)

const (
	// ScannerKeyID is the key ID for the production signing key (2026-05 rotation).
	// Used when VETTCODE_SIGNING_KEY or VETTCODE_SIGNING_KEY_FILE is set.
	ScannerKeyID = "vettcode-scanner-key-2026-05"

	// devScannerKeyID is the key ID for the dev fallback key.
	// Only used when no production key is configured.
	devScannerKeyID = "vettcode-scanner-key-2026-03"

	// envSigningKey is the environment variable holding the base64-encoded
	// Ed25519 private key seed (32 bytes). Set this in production builds or
	// CI to inject the real signing key.
	envSigningKey = "VETTCODE_SIGNING_KEY"

	// envSigningKeyFile is the environment variable holding a file path to
	// a file containing the raw 32-byte Ed25519 seed. Alternative to
	// VETTCODE_SIGNING_KEY for environments that use file-based secrets
	// (e.g., Kubernetes secrets, Docker secrets at /run/secrets/).
	envSigningKeyFile = "VETTCODE_SIGNING_KEY_FILE"
)

// embeddedSigningKeySeed is injected at build time via:
//
//	-ldflags "-X github.com/vettcode/scanner/internal/output.embeddedSigningKeySeed=<base64>"
//
// When set, it takes priority over VETTCODE_SIGNING_KEY / VETTCODE_SIGNING_KEY_FILE.
// Release builds set this to the production key seed; dev builds leave it empty.
var embeddedSigningKeySeed string

// signingKeySource records where the active key was loaded from (for diagnostics).
var signingKeySource string

// activeKeyID is the key ID that matches the currently loaded private key.
var activeKeyID string

// scannerPrivateKey is the Ed25519 private key for signing scan results.
var scannerPrivateKey ed25519.PrivateKey
var scannerPublicKey ed25519.PublicKey

func init() {
	loadSigningKey()
}

// loadSigningKey tries, in order:
//  1. Embedded seed (injected at build time via ldflags — production releases)
//     If present and valid, options 2–4 are never reached.
//  2. VETTCODE_SIGNING_KEY env var (base64-encoded 32-byte seed — dev override only)
//  3. VETTCODE_SIGNING_KEY_FILE env var (path to file containing raw 32-byte seed)
//  4. Fallback: deterministic dev key (NEVER use in production releases)
func loadSigningKey() {
	// Option 0: seed embedded at build time via ldflags (production binary)
	if embeddedSigningKeySeed != "" {
		seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(embeddedSigningKeySeed))
		if err != nil {
			seed, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(embeddedSigningKeySeed))
		}
		if err == nil && len(seed) == ed25519.SeedSize {
			scannerPrivateKey = ed25519.NewKeyFromSeed(seed)
			scannerPublicKey = scannerPrivateKey.Public().(ed25519.PublicKey)
			signingKeySource = "embedded"
			activeKeyID = ScannerKeyID
			return
		}
		fmt.Fprintf(os.Stderr, "WARN: embedded signing key seed is invalid; falling back\n")
	}

	// Option 1: base64-encoded seed in env var (only reached when no embedded key)
	if encoded := os.Getenv(envSigningKey); encoded != "" {
		seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil {
			// Try RawStdEncoding (no padding)
			seed, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
		}
		if err == nil && len(seed) == ed25519.SeedSize {
			scannerPrivateKey = ed25519.NewKeyFromSeed(seed)
			scannerPublicKey = scannerPrivateKey.Public().(ed25519.PublicKey)
			signingKeySource = "env:" + envSigningKey
			activeKeyID = ScannerKeyID
			return
		}
		// Bad key data — fall through to next option with a stderr warning
		fmt.Fprintf(os.Stderr, "WARN: %s is set but contains invalid key data (expected %d-byte base64 seed); falling back\n",
			envSigningKey, ed25519.SeedSize)
	}

	// Option 2: seed read from a file
	if keyPath := os.Getenv(envSigningKeyFile); keyPath != "" {
		seed, err := os.ReadFile(keyPath)
		if err == nil && len(seed) == ed25519.SeedSize {
			scannerPrivateKey = ed25519.NewKeyFromSeed(seed)
			scannerPublicKey = scannerPrivateKey.Public().(ed25519.PublicKey)
			signingKeySource = "file:" + keyPath
			activeKeyID = ScannerKeyID
			return
		}
		fmt.Fprintf(os.Stderr, "WARN: %s=%s could not be loaded (err=%v, len=%d); falling back\n",
			envSigningKeyFile, keyPath, err, len(seed))
	}

	// Option 3: deterministic dev key (for development and testing only)
	seed := make([]byte, ed25519.SeedSize)
	copy(seed, []byte("vettcode-dev-signing-key-seed00"))
	scannerPrivateKey = ed25519.NewKeyFromSeed(seed)
	scannerPublicKey = scannerPrivateKey.Public().(ed25519.PublicKey)
	signingKeySource = "dev-fallback"
	activeKeyID = devScannerKeyID
}

// SigningKeySource returns a string describing where the active signing key
// was loaded from: "env:VETTCODE_SIGNING_KEY", "file:/path", or "dev-fallback".
func SigningKeySource() string {
	return signingKeySource
}

// ActiveKeyID returns the key ID that will be embedded in signed scan results.
// Matches the currently loaded private key: ScannerKeyID when a production key
// is configured via VETTCODE_SIGNING_KEY or VETTCODE_SIGNING_KEY_FILE, or the
// dev fallback key ID otherwise.
func ActiveKeyID() string {
	return activeKeyID
}

// IsDevKey returns true if the scanner is using the built-in dev fallback key
// rather than a production key. Useful for printing warnings in release builds.
func IsDevKey() bool {
	return signingKeySource == "dev-fallback"
}

// SignScanResult computes the integrity block for a ScanResult.
// It sets the non-computed integrity fields first (scanner_public_key_id,
// cosigned), then hashes (including nonce if present), then signs.
// The cosign nonce must be set before calling this if co-signing.
func SignScanResult(result *models.ScanResult) error {
	// Preserve any existing nonce (set by co-signing flow)
	existingNonce := result.Integrity.CosignNonce

	// Set non-computed fields before hashing so they're included
	result.Integrity = models.Integrity{
		ScannerPublicKeyID: activeKeyID,
		CosignNonce:        existingNonce,
		Cosigned:           false,
		VerificationLevel:  models.VerificationSelfReported,
	}

	checksum, _, err := CanonicalChecksumForSigning(result)
	if err != nil {
		return fmt.Errorf("compute checksum: %w", err)
	}

	sig := ed25519.Sign(scannerPrivateKey, []byte(checksum))
	result.Integrity.ScanChecksum = checksum
	result.Integrity.ScannerSignature = base64.StdEncoding.EncodeToString(sig)

	return nil
}

// VerifyScannerSignature verifies the scanner's Ed25519 signature
// on a ScanResult. Returns nil if valid, error if invalid.
func VerifyScannerSignature(result *models.ScanResult) error {
	// Recompute the checksum
	checksum, _, err := CanonicalChecksumForSigning(result)
	if err != nil {
		return fmt.Errorf("compute checksum: %w", err)
	}

	if checksum != result.Integrity.ScanChecksum {
		return fmt.Errorf("checksum mismatch: computed %s, got %s", checksum, result.Integrity.ScanChecksum)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(result.Integrity.ScannerSignature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	if !ed25519.Verify(scannerPublicKey, []byte(checksum), sigBytes) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// GetPublicKey returns the scanner's Ed25519 public key (for testing/verification).
func GetPublicKey() ed25519.PublicKey {
	return scannerPublicKey
}
