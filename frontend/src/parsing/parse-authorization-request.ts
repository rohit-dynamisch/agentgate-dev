/**
 * Parses raw JSON into the typed AuthorizationRequest domain model.
 *
 * The frontend does not construct or send authorization requests in G1 (it
 * only displays decisions the backend already made) — but the request
 * shape carries the identity/tool/correlation information the UI needs to
 * show *what was decided about*, so it is modeled and parsed with the same
 * fail-explicit discipline as the result. See AG-UI-G1-02's required
 * semantics: "identity/tool information; correlation ID".
 */

import type { ApiError } from "../models/api-error.js";
import type {
  AttributeValue,
  AttributeValueWire,
  AuthorizationRequest,
  AuthorizationRequestWire,
} from "../models/decision.js";
import type { ParseResult } from "./parse-authorization-result.js";

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

function parseAttributeValue(raw: unknown): AttributeValue | undefined {
  if (!isRecord(raw)) return undefined;
  const w = raw as Partial<AttributeValueWire>;
  if (w.type === "string" && typeof w.value === "string") return { kind: "string", value: w.value };
  if (w.type === "int" && typeof w.value === "number") return { kind: "int", value: w.value };
  if (w.type === "bool" && typeof w.value === "boolean") return { kind: "bool", value: w.value };
  return undefined;
}

export function parseAuthorizationRequest(raw: unknown): ParseResult<AuthorizationRequest> {
  if (!isRecord(raw)) {
    return { ok: false, error: { kind: "invalid_response", message: "request body is not a JSON object" } };
  }
  const wire = raw as Partial<AuthorizationRequestWire>;

  if (typeof wire.execution_id !== "string" || wire.execution_id === "") {
    return { ok: false, error: { kind: "invalid_response", message: `"execution_id" is missing or empty` } };
  }
  if (typeof wire.workspace_id !== "string" || wire.workspace_id === "") {
    return { ok: false, error: { kind: "invalid_response", message: `"workspace_id" is missing or empty` } };
  }
  if (!isRecord(wire.identity) || typeof wire.identity.agent_id !== "string" || wire.identity.agent_id === "") {
    return { ok: false, error: { kind: "invalid_response", message: `"identity.agent_id" is missing or empty` } };
  }
  const roles = wire.identity.roles;
  if (!Array.isArray(roles) || roles.length === 0 || roles.some((r) => typeof r !== "string" || r === "")) {
    return { ok: false, error: { kind: "invalid_response", message: `"identity.roles" must be a non-empty array of non-empty strings` } };
  }
  if (!isRecord(wire.tool) || typeof wire.tool.backend_id !== "string" || wire.tool.backend_id === "" || typeof wire.tool.name !== "string" || wire.tool.name === "") {
    return { ok: false, error: { kind: "invalid_response", message: `"tool.backend_id"/"tool.name" are missing or empty` } };
  }
  if (!isRecord(wire.classification) || typeof wire.classification.known !== "boolean") {
    return { ok: false, error: { kind: "invalid_response", message: `"classification.known" is missing or not a boolean` } };
  }

  const args: Record<string, AttributeValue> = {};
  if (wire.arguments !== undefined) {
    if (!isRecord(wire.arguments)) {
      return { ok: false, error: { kind: "invalid_response", message: `"arguments" must be an object when present` } };
    }
    for (const [key, value] of Object.entries(wire.arguments)) {
      const parsed = parseAttributeValue(value);
      if (parsed === undefined) {
        return { ok: false, error: { kind: "invalid_response", message: `argument "${key}" has an invalid or mismatched type/value` } };
      }
      args[key] = parsed;
    }
  }

  const onBehalfOf = typeof wire.identity.on_behalf_of === "string" ? wire.identity.on_behalf_of : "";
  const risk = typeof wire.classification.risk === "string" ? wire.classification.risk : "";

  return {
    ok: true,
    value: {
      executionId: wire.execution_id,
      workspaceId: wire.workspace_id,
      identity: {
        agentId: wire.identity.agent_id,
        onBehalfOf,
        roles: roles as string[],
      },
      tool: { backendId: wire.tool.backend_id, name: wire.tool.name },
      classification: { known: wire.classification.known, risk },
      arguments: args,
    },
  };
}
