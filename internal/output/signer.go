package output

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"github.com/vettcode/scanner/pkg/models"
)

const (
	// ScannerKeyID identifies the embedded signing key pair.
	// Rotated with each major scanner release.
	ScannerKeyID = "vettcode-scanner-key-2026-03"
)

// scannerPrivateKey is the embedded Ed25519 private key for signing scan results.
// In production, this would be obfuscated and rotated per major release.
// This is a development key — the production key is injected at build time.
var scannerPrivateKey ed25519.PrivateKey
var scannerPublicKey ed25519.PublicKey

func init() {
	// Generate a deterministic dev key from a seed.
	// Production builds replace this with the real key via ldflags or go:embed.
	seed := make([]byte, ed25519.SeedSize)
	copy(seed, []byte("vettcode-dev-signing-key-seed00"))
	scannerPrivateKey = ed25519.NewKeyFromSeed(seed)
	scannerPublicKey = scannerPrivateKey.Public().(ed25519.PublicKey)
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
		ScannerPublicKeyID: ScannerKeyID,
		CosignNonce:        existingNonce,
		Cosigned:           false,
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
