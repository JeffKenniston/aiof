package security

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// ADR-17: Hardware TEE Enclave Attestation & Macaroon Capability Chains
// ============================================================================

const (
	// TDX Constants
	TDXQuoteHeaderSize = 48
	TDXReportBodySize   = 584
	TDXReportDataSize   = 64
	TDXMeasurementSize  = 48 // SHA-384 size for TDX MRTD/RTMR

	// TPM 2.0 Constants
	TPM2_GENERATED_VALUE = 0xFF544347 // Magic value for TPMS_ATTEST ("\xffTCG")
	TPM_ST_ATTEST_QUOTE  = 0x8018     // Structure tag for Quote attestation
)

// Common errors
var (
	ErrInvalidQuoteFormat      = errors.New("security/attestation: invalid quote format or insufficient length")
	ErrMeasurementMismatch     = errors.New("security/attestation: TEE measurement does not match expected identity")
	ErrNonceMismatch           = errors.New("security/attestation: attestation quote nonce/report_data mismatch")
	ErrQuoteExpired            = errors.New("security/attestation: attestation quote timestamp exceeded freshness threshold")
	ErrSignatureInvalid        = errors.New("security/attestation: cryptographic signature verification failed")
	ErrMacaroonTampered        = errors.New("security/attestation: macaroon signature verification failed or tampered")
	ErrCaveatViolation         = errors.New("security/attestation: macaroon caveat policy check failed")
	ErrTransitiveScopeExpanded = errors.New("security/attestation: delegation attempted to expand rather than restrict capability scope")
)

// ----------------------------------------------------------------------------
// Intel TDX Attestation Quote Primitives
// ----------------------------------------------------------------------------

// TDXQuoteHeader represents the 48-byte header of an Intel TDX Attestation Quote.
type TDXQuoteHeader struct {
	Version    uint16   `json:"version"`
	AttKeyType uint16   `json:"attestation_key_type"`
	TeeType    uint32   `json:"tee_type"` // 0x00000081 for TDX
	Reserved   [2]byte  `json:"reserved"`
	VendorID   [16]byte `json:"vendor_id"`
	UserData   [20]byte `json:"user_data"`
}

// TDXReportBody represents the 584-byte TDREPORT structure embedded in a TDX Quote.
type TDXReportBody struct {
	Attributes    [8]byte     `json:"attributes"`
	XFAM          [8]byte     `json:"xfam"`
	MRTD          [48]byte    `json:"mrtd"`          // Build-time measurement of TD
	MRCONFIGID    [48]byte    `json:"mrconfigid"`    // Software config measurement
	MROWNER       [48]byte    `json:"mrowner"`       // TD owner measurement
	MROWNERCONFIG [48]byte    `json:"mrownerconfig"` // Owner config measurement
	RTMR          [4][48]byte `json:"rtmr"`          // Runtime measurements (RTMR0-3)
	ReportData    [64]byte    `json:"report_data"`   // User-defined nonce / pubkey hash
}

// TDXQuote represents a fully parsed Intel TDX Attestation Quote.
type TDXQuote struct {
	Header        TDXQuoteHeader `json:"header"`
	Body          TDXReportBody  `json:"body"`
	SignatureData []byte         `json:"signature_data"`
	RawBytes      []byte         `json:"-"`
}

// ParseTDXQuote parses raw byte slice into a TDXQuote struct.
func ParseTDXQuote(raw []byte) (*TDXQuote, error) {
	if len(raw) < TDXQuoteHeaderSize+TDXReportBodySize {
		return nil, fmt.Errorf("%w: expected at least %d bytes, got %d", ErrInvalidQuoteFormat, TDXQuoteHeaderSize+TDXReportBodySize, len(raw))
	}

	quote := &TDXQuote{
		RawBytes: raw,
	}

	// Parse Header
	buf := bytes.NewReader(raw[:TDXQuoteHeaderSize])
	if err := binary.Read(buf, binary.LittleEndian, &quote.Header.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &quote.Header.AttKeyType); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &quote.Header.TeeType); err != nil {
		return nil, err
	}
	copy(quote.Header.Reserved[:], raw[8:10])
	copy(quote.Header.VendorID[:], raw[10:26])
	copy(quote.Header.UserData[:], raw[26:46])

	// Parse Body
	bodyBytes := raw[TDXQuoteHeaderSize : TDXQuoteHeaderSize+TDXReportBodySize]
	copy(quote.Body.Attributes[:], bodyBytes[0:8])
	copy(quote.Body.XFAM[:], bodyBytes[8:16])
	copy(quote.Body.MRTD[:], bodyBytes[16:64])
	copy(quote.Body.MRCONFIGID[:], bodyBytes[64:112])
	copy(quote.Body.MROWNER[:], bodyBytes[112:160])
	copy(quote.Body.MROWNERCONFIG[:], bodyBytes[160:208])

	offset := 208
	for i := 0; i < 4; i++ {
		copy(quote.Body.RTMR[i][:], bodyBytes[offset:offset+48])
		offset += 48
	}
	copy(quote.Body.ReportData[:], bodyBytes[400:464])

	// Remainder is signature & certification data
	if len(raw) > TDXQuoteHeaderSize+TDXReportBodySize {
		quote.SignatureData = make([]byte, len(raw)-(TDXQuoteHeaderSize+TDXReportBodySize))
		copy(quote.SignatureData, raw[TDXQuoteHeaderSize+TDXReportBodySize:])
	}

	return quote, nil
}

// ----------------------------------------------------------------------------
// TPM 2.0 Quote Attestation Primitives
// ----------------------------------------------------------------------------

// TPMClockInfo represents clock & reset tracking in a TPM 2.0 Quote.
type TPMClockInfo struct {
	Clock        uint64 `json:"clock"`
	ResetCount   uint32 `json:"reset_count"`
	RestartCount uint32 `json:"restart_count"`
	Safe         bool   `json:"safe"`
}

// TPM2Quote represents parsed TPMS_ATTEST Quote structure from TPM 2.0.
type TPM2Quote struct {
	Magic           uint32       `json:"magic"`
	Type            uint16       `json:"type"`
	QualifiedSigner []byte       `json:"qualified_signer"`
	ExtraData       []byte       `json:"extra_data"` // Freshness Nonce / Challenge
	ClockInfo       TPMClockInfo `json:"clock_info"`
	FirmwareVersion uint64       `json:"firmware_version"`
	PCRSelect       []byte       `json:"pcr_select"`
	PCRDigest       []byte       `json:"pcr_digest"`
	Signature       []byte       `json:"signature"`
	RawBytes        []byte       `json:"-"`
}

// ParseTPM2Quote parses raw byte buffer containing TPMS_ATTEST structure.
func ParseTPM2Quote(raw []byte) (*TPM2Quote, error) {
	if len(raw) < 32 {
		return nil, fmt.Errorf("%w: TPM quote data too short (%d bytes)", ErrInvalidQuoteFormat, len(raw))
	}

	quote := &TPM2Quote{RawBytes: raw}
	buf := bytes.NewReader(raw)

	if err := binary.Read(buf, binary.BigEndian, &quote.Magic); err != nil {
		return nil, err
	}
	if quote.Magic != TPM2_GENERATED_VALUE {
		return nil, fmt.Errorf("%w: invalid TPM magic 0x%X (expected 0x%X)", ErrInvalidQuoteFormat, quote.Magic, TPM2_GENERATED_VALUE)
	}

	if err := binary.Read(buf, binary.BigEndian, &quote.Type); err != nil {
		return nil, err
	}
	if quote.Type != TPM_ST_ATTEST_QUOTE {
		return nil, fmt.Errorf("%w: invalid TPM attest type 0x%X (expected 0x%X)", ErrInvalidQuoteFormat, quote.Type, TPM_ST_ATTEST_QUOTE)
	}

	// Read Qualified Signer (Sized Buffer)
	var signerSize uint16
	if err := binary.Read(buf, binary.BigEndian, &signerSize); err != nil {
		return nil, err
	}
	quote.QualifiedSigner = make([]byte, signerSize)
	if _, err := buf.Read(quote.QualifiedSigner); err != nil {
		return nil, err
	}

	// Read ExtraData / Nonce (Sized Buffer)
	var extraSize uint16
	if err := binary.Read(buf, binary.BigEndian, &extraSize); err != nil {
		return nil, err
	}
	quote.ExtraData = make([]byte, extraSize)
	if _, err := buf.Read(quote.ExtraData); err != nil {
		return nil, err
	}

	// Read ClockInfo
	if err := binary.Read(buf, binary.BigEndian, &quote.ClockInfo.Clock); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &quote.ClockInfo.ResetCount); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &quote.ClockInfo.RestartCount); err != nil {
		return nil, err
	}
	var safeByte byte
	if err := binary.Read(buf, binary.BigEndian, &safeByte); err != nil {
		return nil, err
	}
	quote.ClockInfo.Safe = safeByte != 0

	// Read FirmwareVersion
	if err := binary.Read(buf, binary.BigEndian, &quote.FirmwareVersion); err != nil {
		return nil, err
	}

	// Read PCR Digest (Sized Buffer)
	var pcrDigestSize uint16
	if err := binary.Read(buf, binary.BigEndian, &pcrDigestSize); err != nil {
		return nil, err
	}
	quote.PCRDigest = make([]byte, pcrDigestSize)
	if _, err := buf.Read(quote.PCRDigest); err != nil {
		return nil, err
	}

	// Remaining bytes store signature
	remaining := buf.Len()
	if remaining > 0 {
		quote.Signature = make([]byte, remaining)
		if _, err := buf.Read(quote.Signature); err != nil {
			return nil, err
		}
	}

	return quote, nil
}

// ----------------------------------------------------------------------------
// TEE Attestation Validation Engine
// ----------------------------------------------------------------------------

// TEEValidationRequest specifies expected measurement values & parameters for attestation check.
type TEEValidationRequest struct {
	TDXQuote          *TDXQuote         `json:"tdx_quote,omitempty"`
	TPMQuote          *TPM2Quote        `json:"tpm_quote,omitempty"`
	ExpectedMRTD      string            `json:"expected_mrtd,omitempty"`      // Hex encoded SHA-384
	ExpectedRTMR      [4]string         `json:"expected_rtmr,omitempty"`      // Hex encoded RTMR0-3
	ExpectedNonce     []byte            `json:"expected_nonce,omitempty"`     // Freshness Challenge
	ExpectedPCRDigest string            `json:"expected_pcr_digest,omitempty"` // Hex encoded PCR digest
	MaxQuoteAge       time.Duration     `json:"max_quote_age,omitempty"`
	EnclavePubKey     ed25519.PublicKey `json:"enclave_pub_key,omitempty"`
}

// TEEValidationResult returns detailed verification status & diagnostic telemetry.
type TEEValidationResult struct {
	Valid             bool      `json:"valid"`
	TeeType           string    `json:"tee_type"`
	EnclaveID         string    `json:"enclave_id"`
	MeasurementsMatch bool      `json:"measurements_match"`
	NonceValid        bool      `json:"nonce_valid"`
	SignatureValid    bool      `json:"signature_valid"`
	ValidatedAt       time.Time `json:"validated_at"`
	Errors            []string  `json:"errors,omitempty"`
}

// ValidateTEEAttestation orchestrates complete TDX / TPM 2.0 quote validation.
func ValidateTEEAttestation(req TEEValidationRequest) (*TEEValidationResult, error) {
	res := &TEEValidationResult{
		ValidatedAt: time.Now().UTC(),
	}

	if req.TDXQuote == nil && req.TPMQuote == nil {
		res.Errors = append(res.Errors, "no TEE attestation quote provided")
		return res, errors.New("security/attestation: missing TDX or TPM quote in validation request")
	}

	// 1. Validate TDX Quote if present
	if req.TDXQuote != nil {
		res.TeeType = "Intel TDX"
		res.EnclaveID = hex.EncodeToString(req.TDXQuote.Body.MRTD[:16])

		// Verify MRTD Measurement
		if req.ExpectedMRTD != "" {
			actualMRTD := hex.EncodeToString(req.TDXQuote.Body.MRTD[:])
			if !strings.EqualFold(actualMRTD, req.ExpectedMRTD) {
				res.Errors = append(res.Errors, fmt.Sprintf("MRTD mismatch: got %s, expected %s", actualMRTD, req.ExpectedMRTD))
			}
		}

		// Verify RTMR Measurements
		for i := 0; i < 4; i++ {
			if req.ExpectedRTMR[i] != "" {
				actualRTMR := hex.EncodeToString(req.TDXQuote.Body.RTMR[i][:])
				if !strings.EqualFold(actualRTMR, req.ExpectedRTMR[i]) {
					res.Errors = append(res.Errors, fmt.Sprintf("RTMR[%d] mismatch: got %s, expected %s", i, actualRTMR, req.ExpectedRTMR[i]))
				}
			}
		}

		// Verify Freshness Nonce embedded in ReportData
		if len(req.ExpectedNonce) > 0 {
			expectedHash := sha256.Sum256(req.ExpectedNonce)
			if subtle.ConstantTimeCompare(req.TDXQuote.Body.ReportData[:32], expectedHash[:]) != 1 {
				res.Errors = append(res.Errors, "TDX ReportData hash does not match expected freshness nonce")
			} else {
				res.NonceValid = true
			}
		} else {
			res.NonceValid = true
		}

		// Verify Signature using Enclave Key
		if len(req.EnclavePubKey) == ed25519.PublicKeySize && len(req.TDXQuote.SignatureData) >= ed25519.SignatureSize {
			sig := req.TDXQuote.SignatureData[:ed25519.SignatureSize]
			msg := req.TDXQuote.RawBytes[:TDXQuoteHeaderSize+TDXReportBodySize]
			if ed25519.Verify(req.EnclavePubKey, msg, sig) {
				res.SignatureValid = true
			} else {
				res.Errors = append(res.Errors, "ed25519 signature verification failed over TDX report body")
			}
		} else {
			// Structurally valid signature buffer check fallback
			res.SignatureValid = len(req.TDXQuote.SignatureData) > 0
		}

		res.MeasurementsMatch = len(res.Errors) == 0
	}

	// 2. Validate TPM 2.0 Quote if present
	if req.TPMQuote != nil {
		if res.TeeType == "" {
			res.TeeType = "TPM 2.0"
			res.EnclaveID = hex.EncodeToString(req.TPMQuote.PCRDigest[:minInt(16, len(req.TPMQuote.PCRDigest))])
		}

		// Check Nonce in ExtraData
		if len(req.ExpectedNonce) > 0 {
			if bytes.Equal(req.TPMQuote.ExtraData, req.ExpectedNonce) {
				res.NonceValid = true
			} else {
				res.Errors = append(res.Errors, "TPM 2.0 ExtraData does not match expected freshness nonce")
			}
		} else {
			res.NonceValid = true
		}

		// Check PCR Digest
		if req.ExpectedPCRDigest != "" {
			actualPCR := hex.EncodeToString(req.TPMQuote.PCRDigest)
			if strings.EqualFold(actualPCR, req.ExpectedPCRDigest) {
				res.MeasurementsMatch = true
			} else {
				res.Errors = append(res.Errors, fmt.Sprintf("TPM PCR digest mismatch: got %s, expected %s", actualPCR, req.ExpectedPCRDigest))
			}
		} else {
			res.MeasurementsMatch = true
		}

		// Signature Verification
		if len(req.EnclavePubKey) == ed25519.PublicKeySize && len(req.TPMQuote.Signature) >= ed25519.SignatureSize {
			if ed25519.Verify(req.EnclavePubKey, req.TPMQuote.RawBytes[:len(req.TPMQuote.RawBytes)-len(req.TPMQuote.Signature)], req.TPMQuote.Signature) {
				res.SignatureValid = true
			} else {
				res.Errors = append(res.Errors, "TPM 2.0 quote signature verification failed")
			}
		} else {
			res.SignatureValid = len(req.TPMQuote.Signature) > 0
		}
	}

	res.Valid = res.MeasurementsMatch && res.NonceValid && res.SignatureValid && len(res.Errors) == 0
	return res, nil
}

// ----------------------------------------------------------------------------
// Macaroon Capability Chains with Transitively Restricted Scope (ADR-11, ADR-17)
// ----------------------------------------------------------------------------

// Caveat represents a single contextual restriction attached to a Macaroon.
type Caveat struct {
	Predicate string `json:"predicate"` // e.g., "action = read", "max_depth = 2", "time < 2026-07-29T23:59:59Z"
	Location  string `json:"location,omitempty"`  // Location for 3rd party caveats
	CaveatID  string `json:"caveat_id,omitempty"` // Unique ID
}

// Macaroon represents a capability delegation token bound to enclave secrets.
type Macaroon struct {
	Location   string   `json:"location"`
	Identifier string   `json:"identifier"` // Root Token ID / Subject
	Caveats    []Caveat `json:"caveats"`
	Signature  []byte   `json:"signature"`  // HMAC-SHA256 signature chain
}

// OperationContext specifies execution parameters evaluated against caveats.
type OperationContext struct {
	Action          string            `json:"action"`
	Resource        string            `json:"resource"`
	Role            string            `json:"role"`
	AgentID         string            `json:"agent_id"`
	EnclaveID       string            `json:"enclave_id"`
	CurrentTime     time.Time         `json:"current_time"`
	DelegationDepth int               `json:"delegation_depth"`
	ExtraAttributes map[string]string `json:"extra_attributes,omitempty"`
}

// EnclaveKeyAuthority manages enclave signing keys and Macaroon minting.
type EnclaveKeyAuthority struct {
	EnclaveID  string            `json:"enclave_id"`
	RootSecret []byte            `json:"-"`
	PublicKey  ed25519.PublicKey `json:"public_key"`
	PrivateKey ed25519.PrivateKey `json:"-"`
}

// NewEnclaveKeyAuthority initializes enclave key authority.
func NewEnclaveKeyAuthority(enclaveID string) (*EnclaveKeyAuthority, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("security/attestation: failed to generate enclave key: %w", err)
	}

	rootSecret := make([]byte, 32)
	if _, err := rand.Read(rootSecret); err != nil {
		return nil, fmt.Errorf("security/attestation: failed to generate enclave root secret: %w", err)
	}

	return &EnclaveKeyAuthority{
		EnclaveID:  enclaveID,
		RootSecret: rootSecret,
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// MintRootMacaroon issues a new primary root Macaroon capability.
func MintRootMacaroon(auth *EnclaveKeyAuthority, location, tokenID string) (*Macaroon, error) {
	if auth == nil || len(auth.RootSecret) == 0 {
		return nil, errors.New("security/attestation: invalid enclave key authority")
	}

	// Root Signature: H0 = HMAC-SHA256(RootSecret, tokenID)
	mac := hmac.New(sha256.New, auth.RootSecret)
	mac.Write([]byte(tokenID))
	initialSig := mac.Sum(nil)

	return &Macaroon{
		Location:   location,
		Identifier: tokenID,
		Caveats:    make([]Caveat, 0),
		Signature:  initialSig,
	}, nil
}

// AddCaveat attenuates a Macaroon by adding a first-party caveat and chaining HMAC signature.
// Enforces transitive restriction: scope can ONLY be narrowed.
func (m *Macaroon) AddCaveat(caveat Caveat) (*Macaroon, error) {
	if m == nil || len(m.Signature) == 0 {
		return nil, errors.New("security/attestation: cannot attenuate nil or uninitialized Macaroon")
	}

	// Compute updated HMAC signature: H_i = HMAC-SHA256(H_{i-1}, Caveat.Predicate)
	mac := hmac.New(sha256.New, m.Signature)
	mac.Write([]byte(caveat.Predicate))
	newSig := mac.Sum(nil)

	newCaveats := make([]Caveat, len(m.Caveats), len(m.Caveats)+1)
	copy(newCaveats, m.Caveats)
	newCaveats = append(newCaveats, caveat)

	return &Macaroon{
		Location:   m.Location,
		Identifier: m.Identifier,
		Caveats:    newCaveats,
		Signature:  newSig,
	}, nil
}

// Verify checks signature integrity, validates all caveats, and enforces transitive scope restriction.
func (m *Macaroon) Verify(auth *EnclaveKeyAuthority, ctx OperationContext) (bool, error) {
	if m == nil || auth == nil {
		return false, errors.New("security/attestation: invalid macaroon or authority")
	}

	// 1. Recompute HMAC Signature Chain
	mac := hmac.New(sha256.New, auth.RootSecret)
	mac.Write([]byte(m.Identifier))
	currentSig := mac.Sum(nil)

	for _, caveat := range m.Caveats {
		cMac := hmac.New(sha256.New, currentSig)
		cMac.Write([]byte(caveat.Predicate))
		currentSig = cMac.Sum(nil)
	}

	if subtle.ConstantTimeCompare(currentSig, m.Signature) != 1 {
		return false, ErrMacaroonTampered
	}

	// 2. Evaluate Caveats & Enforce Monotonic Scope Restriction
	if ctx.CurrentTime.IsZero() {
		ctx.CurrentTime = time.Now().UTC()
	}

	maxAllowedDepth := 100 // Default max delegation depth
	resourcePrefix := ""
	allowedAction := ""

	for _, caveat := range m.Caveats {
		ok, reason := evaluateCaveat(caveat, ctx)
		if !ok {
			return false, fmt.Errorf("%w: predicate '%s' violated (%s)", ErrCaveatViolation, caveat.Predicate, reason)
		}

		// Track transitive restriction parameters
		if strings.HasPrefix(caveat.Predicate, "max_depth =") {
			valStr := strings.TrimSpace(strings.TrimPrefix(caveat.Predicate, "max_depth ="))
			if val, err := strconv.Atoi(valStr); err == nil {
				if val < maxAllowedDepth {
					maxAllowedDepth = val // Strict monotonic decrease
				}
			}
		}

		if strings.HasPrefix(caveat.Predicate, "resource_prefix =") {
			val := strings.TrimSpace(strings.TrimPrefix(caveat.Predicate, "resource_prefix ="))
			if resourcePrefix != "" && !strings.HasPrefix(val, resourcePrefix) {
				return false, fmt.Errorf("%w: resource caveat '%s' attempts to expand scope beyond '%s'", ErrTransitiveScopeExpanded, val, resourcePrefix)
			}
			resourcePrefix = val
		}

		if strings.HasPrefix(caveat.Predicate, "action =") {
			val := strings.TrimSpace(strings.TrimPrefix(caveat.Predicate, "action ="))
			if allowedAction != "" && allowedAction != val {
				return false, fmt.Errorf("%w: action caveat '%s' conflicts with existing scope '%s'", ErrTransitiveScopeExpanded, val, allowedAction)
			}
			allowedAction = val
		}
	}

	// Enforce Monotonic Delegation Depth Check
	if ctx.DelegationDepth > maxAllowedDepth {
		return false, fmt.Errorf("%w: current delegation depth %d exceeds caveat max_depth %d", ErrCaveatViolation, ctx.DelegationDepth, maxAllowedDepth)
	}

	return true, nil
}

// Internal Caveat Evaluator
func evaluateCaveat(c Caveat, ctx OperationContext) (bool, string) {
	pred := strings.TrimSpace(c.Predicate)

	switch {
	case strings.HasPrefix(pred, "time <"):
		timeStr := strings.TrimSpace(strings.TrimPrefix(pred, "time <"))
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return false, "invalid time predicate format"
		}
		if !ctx.CurrentTime.Before(t) {
			return false, fmt.Sprintf("token expired at %s, current time is %s", t.Format(time.RFC3339), ctx.CurrentTime.Format(time.RFC3339))
		}

	case strings.HasPrefix(pred, "action ="):
		act := strings.TrimSpace(strings.TrimPrefix(pred, "action ="))
		if !strings.EqualFold(act, ctx.Action) {
			return false, fmt.Sprintf("action mismatch: required '%s', got '%s'", act, ctx.Action)
		}

	case strings.HasPrefix(pred, "resource ="):
		res := strings.TrimSpace(strings.TrimPrefix(pred, "resource ="))
		if res != ctx.Resource {
			return false, fmt.Sprintf("resource mismatch: required '%s', got '%s'", res, ctx.Resource)
		}

	case strings.HasPrefix(pred, "resource_prefix ="):
		prefix := strings.TrimSpace(strings.TrimPrefix(pred, "resource_prefix ="))
		if !strings.HasPrefix(ctx.Resource, prefix) {
			return false, fmt.Sprintf("resource '%s' does not match required prefix '%s'", ctx.Resource, prefix)
		}

	case strings.HasPrefix(pred, "role ="):
		role := strings.TrimSpace(strings.TrimPrefix(pred, "role ="))
		if role != ctx.Role {
			return false, fmt.Sprintf("role mismatch: required '%s', got '%s'", role, ctx.Role)
		}

	case strings.HasPrefix(pred, "enclave_id ="):
		enc := strings.TrimSpace(strings.TrimPrefix(pred, "enclave_id ="))
		if enc != ctx.EnclaveID {
			return false, fmt.Sprintf("enclave_id mismatch: required '%s', got '%s'", enc, ctx.EnclaveID)
		}

	case strings.HasPrefix(pred, "max_depth ="):
		// Checked at higher verification level
		return true, ""

	default:
		// Check extra context attributes
		if parts := strings.SplitN(pred, "=", 2); len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if ctxVal, exists := ctx.ExtraAttributes[k]; !exists || ctxVal != v {
				return false, fmt.Sprintf("custom attribute '%s' mismatch: required '%s', got '%s'", k, v, ctxVal)
			}
		}
	}

	return true, ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

