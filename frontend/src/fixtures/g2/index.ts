/**
 * G2 Cross-Workstream Governance Fixtures
 *
 * Deterministic test fixtures mirroring Go backend contracts:
 * - internal/identity (MapperConfig, MappedIdentity, FailureClass)
 * - internal/toolregistry (ToolID, GovernanceRecord, SchemaFingerprint, DriftStatus)
 * - internal/argdecl (ArgType, Declaration, ResolvedArg)
 */

import identityFixtures from "./identity.json" with { type: "json" };
import toolGovernanceFixtures from "./tool-governance.json" with { type: "json" };
import argumentDeclarationFixtures from "./argument-declaration.json" with { type: "json" };

export const g2Fixtures = {
  identity: identityFixtures,
  toolGovernance: toolGovernanceFixtures,
  argumentDeclaration: argumentDeclarationFixtures,
} as const;
