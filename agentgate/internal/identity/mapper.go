// Package identity resolves verified request identity (agent,
// on-behalf-of subject, roles) from gateway-validated claims into the
// context AgentGate's policy engine evaluates against.
//
// # Design contract
//
// This package does NOT validate or verify tokens. It receives a
// map of pre-validated claims provided by the gateway's JWT authentication
// layer and maps named fields from that map into AgentGate's typed
// [MappedIdentity] according to a [MapperConfig]. Claim names are
// configuration-driven; none are hardcoded.
//
// # Failure classes
//
// The package distinguishes four distinct failure cases, all of which
// fail closed — producing a non-nil [MappingError] rather than a
// partially-populated identity:
//
//   - [ErrMissingClaim]: a required claim key is absent from the claims map.
//   - [ErrMalformedClaim]: a claim key is present but its value is not a
//     non-empty string.
//   - [ErrAmbiguousIdentity]: conflicting identity signals are present
//     (e.g. on_behalf_of equals agent_id).
//   - [ErrMissingRoles]: the roles claim is present but resolved to an empty
//     set, or every role string is blank.
//
// Open decision O-001 (downstream credential propagation) is deliberately
// not addressed here. This package owns only the inbound identity boundary.
package identity

import (
	"errors"
	"fmt"
	"strings"
)

// FailureClass is the enumerated set of identity-mapping failure modes.
// Every failure fails closed (denies); the class tells audit/operators why.
type FailureClass string

const (
	// ErrMissingClaim: a required claim key is absent from the input map.
	ErrMissingClaim FailureClass = "missing_claim"

	// ErrMalformedClaim: a claim key is present but its value is not a
	// usable non-empty string.
	ErrMalformedClaim FailureClass = "malformed_claim"

	// ErrAmbiguousIdentity: conflicting identity signals are present
	// (e.g. on_behalf_of equals agent_id when it must differ).
	ErrAmbiguousIdentity FailureClass = "ambiguous_identity"

	// ErrMissingRoles: the roles claim resolved to an empty or all-blank
	// set; an identity with no usable roles cannot be authorized.
	ErrMissingRoles FailureClass = "missing_roles"
)

// MappingError is returned by [Mapper.Map] for every failure case.
// It carries the failure class (for programmatic handling) and a
// human-readable detail (for logs/audit). It is never nil on failure and
// never returned on success.
type MappingError struct {
	Class  FailureClass
	Detail string
}

func (e *MappingError) Error() string {
	return fmt.Sprintf("identity mapping: %s: %s", e.Class, e.Detail)
}

// Is supports errors.Is matching on FailureClass values stored as
// sentinel errors via [AsClass].
func (e *MappingError) Is(target error) bool {
	if t, ok := target.(*MappingError); ok {
		return e.Class == t.Class
	}
	return false
}

// AsClass returns a sentinel *MappingError for use with errors.Is.
//
//	errors.Is(err, identity.AsClass(identity.ErrMissingClaim))
func AsClass(c FailureClass) error {
	return &MappingError{Class: c}
}

// RolesSeparator is the separator used when roles are packed into a single
// claim string. When roles arrive as a space-separated string (a common JWT
// convention), splitting on this separator produces the role list.
// When the claim is already modelled as a repeated/multi-value header, the
// gateway layer is expected to join values with this separator before
// passing them to Map.
//
// The separator is a single space, matching the OAuth2 scope convention
// common in JWT "scope" or "roles" claims. Configuring a multi-value claim
// as a pre-split []string is NOT supported at this boundary — all claims
// arrive as raw strings (the gateway's view of a header or JWT field).
const RolesSeparator = " "

// MapperConfig specifies which claim keys to read for each identity field.
// Claim names are deployment-specific and must never be hardcoded in
// AgentGate logic (docs/TECH_STACK.md §2.4, OPEN_DECISIONS.md O-001).
//
// All fields are required unless documented as optional. Validation is
// performed at [NewMapper] time, not per-request, so an operator
// misconfiguration fails at startup rather than at runtime.
type MapperConfig struct {
	// AgentIDClaim is the claim key whose value becomes [MappedIdentity.AgentID].
	// Required. The value must be a non-empty string.
	AgentIDClaim string

	// OnBehalfOfClaim is the claim key for the delegated human identity.
	// Optional: when empty the field is never read and
	// [MappedIdentity.OnBehalfOf] is always "".
	OnBehalfOfClaim string

	// RolesClaim is the claim key whose value becomes the roles list.
	// The value must be a non-empty space-separated string of role names.
	// Required.
	RolesClaim string
}

// Validate returns an error when the config is not usable. Called by
// [NewMapper]; exposed separately for config-loading code that wants to
// validate before constructing a Mapper.
func (c MapperConfig) Validate() error {
	if strings.TrimSpace(c.AgentIDClaim) == "" {
		return errors.New("identity: MapperConfig.AgentIDClaim must not be empty")
	}
	if strings.TrimSpace(c.RolesClaim) == "" {
		return errors.New("identity: MapperConfig.RolesClaim must not be empty")
	}
	return nil
}

// Mapper maps a gateway-validated claims map into a [MappedIdentity].
// Construct with [NewMapper].
type Mapper struct {
	cfg MapperConfig
}

// NewMapper constructs a Mapper from a validated config. Returns an error
// if the config is invalid (e.g. required claim keys are blank).
func NewMapper(cfg MapperConfig) (*Mapper, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Mapper{cfg: cfg}, nil
}

// MappedIdentity is the result of a successful mapping — a typed,
// validated identity ready to be placed into a [decision.Request].
// It is only returned when all required claims are present and consistent;
// any failure returns a *[MappingError] instead.
type MappedIdentity struct {
	// AgentID is the authoritative identifier for the calling agent.
	AgentID string

	// OnBehalfOf is the human the agent is acting for. Empty string
	// means no delegated identity for this call.
	OnBehalfOf string

	// Roles is the non-empty set of role assignments for this identity.
	// Guaranteed non-empty (failure otherwise).
	Roles []string
}

// Map resolves claims into a [MappedIdentity]. Claims is the
// key→value map of gateway-validated context (e.g. JWT claims already
// verified by agentgateway's JWT authentication layer). Map does not
// validate tokens; it only reads named fields.
//
// All four failure classes ([ErrMissingClaim], [ErrMalformedClaim],
// [ErrAmbiguousIdentity], [ErrMissingRoles]) result in a non-nil
// *[MappingError] return and a zero MappedIdentity.
func (m *Mapper) Map(claims map[string]string) (MappedIdentity, *MappingError) {
	// --- Agent ID (required) ---
	agentID, merr := requireStringClaim(claims, m.cfg.AgentIDClaim)
	if merr != nil {
		return MappedIdentity{}, merr
	}

	// --- OnBehalfOf (optional) ---
	var onBehalfOf string
	if m.cfg.OnBehalfOfClaim != "" {
		raw, present := claims[m.cfg.OnBehalfOfClaim]
		if present {
			trimmed := strings.TrimSpace(raw)
			if trimmed == "" {
				// Present but blank — treat as malformed, not simply absent.
				return MappedIdentity{}, &MappingError{
					Class:  ErrMalformedClaim,
					Detail: fmt.Sprintf("claim %q is present but blank", m.cfg.OnBehalfOfClaim),
				}
			}
			onBehalfOf = trimmed
		}
		// Absent optional claim → onBehalfOf stays "".
	}

	// --- Roles (required) ---
	rawRoles, merr := requireStringClaim(claims, m.cfg.RolesClaim)
	if merr != nil {
		return MappedIdentity{}, merr
	}
	roles := splitRoles(rawRoles)
	if len(roles) == 0 {
		return MappedIdentity{}, &MappingError{
			Class:  ErrMissingRoles,
			Detail: fmt.Sprintf("claim %q resolved to an empty role set", m.cfg.RolesClaim),
		}
	}

	// --- Ambiguity check ---
	if onBehalfOf != "" && onBehalfOf == agentID {
		return MappedIdentity{}, &MappingError{
			Class:  ErrAmbiguousIdentity,
			Detail: "on_behalf_of must not equal agent_id",
		}
	}

	return MappedIdentity{
		AgentID:    agentID,
		OnBehalfOf: onBehalfOf,
		Roles:      roles,
	}, nil
}

// requireStringClaim reads a required claim from the map, returning a
// *MappingError if absent or blank.
func requireStringClaim(claims map[string]string, key string) (string, *MappingError) {
	raw, ok := claims[key]
	if !ok {
		return "", &MappingError{
			Class:  ErrMissingClaim,
			Detail: fmt.Sprintf("required claim %q is absent", key),
		}
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", &MappingError{
			Class:  ErrMalformedClaim,
			Detail: fmt.Sprintf("claim %q is present but blank", key),
		}
	}
	return trimmed, nil
}

// splitRoles splits a space-separated roles string into a deduplicated,
// blank-filtered slice. Returns nil (empty) when no valid roles are found.
func splitRoles(raw string) []string {
	parts := strings.Split(raw, RolesSeparator)
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if _, dup := seen[trimmed]; dup {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
