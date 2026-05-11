package output

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/pkg/models"
)

func newTestScanResult() *models.ScanResult {
	grade := models.GradeB
	return &models.ScanResult{
		Version:        "1.0",
		ScanID:         "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:      "2026-03-13T10:30:00Z",
		ScannerVersion: "0.1.0",
		TotalLOC:       42600,
		TotalFileCount: 350,
		RepoCount:      2,
		Repositories:   []models.Repository{},
		TechStack: models.TechStack{
			Frameworks: []string{"Next.js", "FastAPI"},
		},
		Summary: models.Summary{
			OverallGrade:     &grade,
			ScoredCategories: []string{"security", "maintainability"},
			TopRisks:         []models.Risk{},
			TopStrengths:     []models.Strength{},
		},
		PricingTier: models.PricingTier{
			Tier:   models.PricingTierStandard,
			Reason: "42,600 LOC",
		},
		Warnings: []models.Warning{},
	}
}

func TestSignScanResult(t *testing.T) {
	result := newTestScanResult()

	err := SignScanResult(result)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Integrity.ScanChecksum)
	assert.Equal(t, ActiveKeyID(), result.Integrity.ScannerPublicKeyID)
	assert.NotEmpty(t, result.Integrity.ScannerSignature)
	assert.False(t, result.Integrity.Cosigned)
	assert.Nil(t, result.Integrity.CosignNonce)
	assert.Nil(t, result.Integrity.PlatformCosignature)
	assert.Equal(t, models.VerificationSelfReported, result.Integrity.VerificationLevel,
		"signing should set verification_level to self_reported")
}

func TestSignScanResult_VerifyRoundTrip(t *testing.T) {
	result := newTestScanResult()

	require.NoError(t, SignScanResult(result))
	require.NoError(t, VerifyScannerSignature(result))
}

func TestVerifyScannerSignature_TamperedData(t *testing.T) {
	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	// Tamper with the data
	result.TotalLOC = 999999

	err := VerifyScannerSignature(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestVerifyScannerSignature_TamperedSignature(t *testing.T) {
	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	// Tamper with the signature
	result.Integrity.ScannerSignature = base64.StdEncoding.EncodeToString([]byte("fake-signature-that-is-64-bytes-long-for-ed25519-xxxxxxxxxxxxxx"))

	err := VerifyScannerSignature(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verification failed")
}

func TestSignScanResult_DeterministicChecksum(t *testing.T) {
	r1 := newTestScanResult()
	r2 := newTestScanResult()

	require.NoError(t, SignScanResult(r1))
	require.NoError(t, SignScanResult(r2))

	// Same input → same checksum
	assert.Equal(t, r1.Integrity.ScanChecksum, r2.Integrity.ScanChecksum)
}

func TestSignScanResult_IntegrityExcluded(t *testing.T) {
	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	// Signing again should produce the same checksum (integrity block excluded)
	checksum1 := result.Integrity.ScanChecksum
	require.NoError(t, SignScanResult(result))
	assert.Equal(t, checksum1, result.Integrity.ScanChecksum)
}

func TestGetPublicKey(t *testing.T) {
	pk := GetPublicKey()
	assert.Equal(t, ed25519.PublicKeySize, len(pk))
}

func TestSigningKeySource_DefaultIsDev(t *testing.T) {
	// Without env vars set, should use dev fallback
	assert.Equal(t, "dev-fallback", SigningKeySource())
	assert.True(t, IsDevKey())
}

func TestLoadSigningKey_FromEnv(t *testing.T) {
	// Generate a test key
	seed := make([]byte, ed25519.SeedSize)
	copy(seed, []byte("test-env-key-seed-0000000000"))
	encoded := base64.StdEncoding.EncodeToString(seed)

	// Save original state
	origPriv := scannerPrivateKey
	origPub := scannerPublicKey
	origSource := signingKeySource
	defer func() {
		scannerPrivateKey = origPriv
		scannerPublicKey = origPub
		signingKeySource = origSource
	}()

	t.Setenv("VETTCODE_SIGNING_KEY", encoded)
	t.Setenv("VETTCODE_SIGNING_KEY_FILE", "")
	loadSigningKey()

	assert.Equal(t, "env:VETTCODE_SIGNING_KEY", SigningKeySource())
	assert.False(t, IsDevKey())

	// Key should work for signing
	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))
	require.NoError(t, VerifyScannerSignature(result))
}

func TestLoadSigningKey_FromFile(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	copy(seed, []byte("test-file-key-seed-000000000"))

	tmpFile := t.TempDir() + "/test-key.seed"
	require.NoError(t, os.WriteFile(tmpFile, seed, 0600))

	origPriv := scannerPrivateKey
	origPub := scannerPublicKey
	origSource := signingKeySource
	defer func() {
		scannerPrivateKey = origPriv
		scannerPublicKey = origPub
		signingKeySource = origSource
	}()

	t.Setenv("VETTCODE_SIGNING_KEY", "")
	t.Setenv("VETTCODE_SIGNING_KEY_FILE", tmpFile)
	loadSigningKey()

	assert.Equal(t, "file:"+tmpFile, SigningKeySource())
	assert.False(t, IsDevKey())

	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))
	require.NoError(t, VerifyScannerSignature(result))
}

func TestScannerKeyID_MatchesExpectedFormat(t *testing.T) {
	// Per shared-contracts.md: Key IDs follow "vettcode-{component}-key-YYYY-MM"
	keyIDPattern := regexp.MustCompile(`^vettcode-scanner-key-\d{4}-\d{2}$`)
	assert.Regexp(t, keyIDPattern, ScannerKeyID,
		"ScannerKeyID should match ^vettcode-scanner-key-\\d{4}-\\d{2}$")
	assert.Regexp(t, keyIDPattern, ActiveKeyID(),
		"ActiveKeyID() should match ^vettcode-scanner-key-\\d{4}-\\d{2}$")
}

func TestSignScanResult_VerificationLevelSelfReported(t *testing.T) {
	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	assert.Equal(t, models.VerificationSelfReported, result.Integrity.VerificationLevel,
		"after signing, verification_level must be self_reported")
}
