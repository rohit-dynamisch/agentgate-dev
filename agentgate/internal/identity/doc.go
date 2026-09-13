// Package identity resolves verified request identity (agent,
// on-behalf-of subject, roles) from gateway-validated claims into the
// context AgentGate's policy engine evaluates against.
//
// G2 implementation: see mapper.go for [Mapper], [MapperConfig],
// [MappedIdentity], and the four failure classes.
//
// This package does NOT validate or verify tokens. Claim names are
// configuration-driven (docs/TECH_STACK.md §2.4). Downstream credential
// propagation (O-001) is not addressed here.
package identity
