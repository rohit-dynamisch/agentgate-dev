/**
 * G2 Argument Declaration Models
 *
 * Framework-agnostic TypeScript types mirroring the Go backend's
 * internal/argdecl package (G2-GO-05, commit c6a5acf).
 *
 * Source of truth: agentgate/internal/argdecl/argdecl.go
 *
 * Invariants preserved from Go:
 * - Closed vocabulary of ArgTypes: "string", "int64", "bool".
 * - Whitelist approach: Only declared arguments are policy-relevant.
 *   Undeclared arguments are ignored and never enter policy evaluation.
 * - Explicit JSON null is always rejected for any declared argument
 *   (never coerced to 0, "", or false).
 * - Missing optional arguments are simply omitted; missing required arguments fail.
 * - Duplicate argument declarations are forbidden.
 */

// ---------------------------------------------------------------------------
// Argument types (mirrors argdecl.ArgType in Go)
// ---------------------------------------------------------------------------

export type ArgType = "string" | "int64" | "bool";

export const ALL_ARG_TYPES: readonly ArgType[] = ["string", "int64", "bool"];

export function isValidArgType(t: string): t is ArgType {
  return ALL_ARG_TYPES.includes(t as ArgType);
}

// ---------------------------------------------------------------------------
// Declaration (mirrors argdecl.Declaration in Go)
// ---------------------------------------------------------------------------

export interface ArgumentDeclarationView {
  /** Name of the argument in the tool input. Required, non-empty. */
  name: string;
  /** Expected argument value type. */
  type: ArgType;
  /** Whether the argument must be present in the tool call. */
  required: boolean;
}

// ---------------------------------------------------------------------------
// Resolved argument value (mirrors argdecl.ResolvedArg in Go)
// ---------------------------------------------------------------------------

export interface ResolvedArgView {
  type: ArgType;
  strVal?: string;
  intVal?: number;
  boolVal?: boolean;
}

// ---------------------------------------------------------------------------
// Resolution error and status
// ---------------------------------------------------------------------------

export interface ArgumentResolutionError {
  argName: string;
  reason: string;
}

export type ArgumentResolutionStatus =
  | "ok"
  | "missing_required"
  | "null_rejected"
  | "type_mismatch"
  | "unrecognized_type";

export interface ArgumentResolutionSuccess {
  ok: true;
  resolved: Record<string, ResolvedArgView>;
}

export interface ArgumentResolutionFailure {
  ok: false;
  error: ArgumentResolutionError;
}

export type ArgumentResolutionResult =
  | ArgumentResolutionSuccess
  | ArgumentResolutionFailure;

/**
 * Validates declarations list for duplicates or empty names.
 */
export function validateDeclarationSet(
  decls: readonly ArgumentDeclarationView[]
): { valid: boolean; error?: string } {
  const seen = new Set<string>();
  for (const d of decls) {
    if (!d.name || d.name.trim() === "") {
      return { valid: false, error: "declaration has empty name" };
    }
    if (!isValidArgType(d.type)) {
      return {
        valid: false,
        error: `declaration "${d.name}" has unrecognized type "${d.type}"`,
      };
    }
    if (seen.has(d.name)) {
      return {
        valid: false,
        error: `duplicate declaration for argument "${d.name}"`,
      };
    }
    seen.add(d.name);
  }
  return { valid: true };
}

/**
 * Resolves raw arguments against declarations following exact backend semantics.
 * - Undeclared arguments are omitted.
 * - Missing required arguments fail.
 * - Missing optional arguments are omitted.
 * - Explicit null fails (for required AND optional).
 * - Type mismatches fail.
 */
export function resolveArguments(
  declarations: readonly ArgumentDeclarationView[],
  rawArgs: Record<string, unknown>
): ArgumentResolutionResult {
  const resolved: Record<string, ResolvedArgView> = {};

  for (const decl of declarations) {
    const hasKey = Object.prototype.hasOwnProperty.call(rawArgs, decl.name);
    const rawVal = rawArgs[decl.name];

    if (!hasKey || rawVal === undefined) {
      if (decl.required) {
        return {
          ok: false,
          error: {
            argName: decl.name,
            reason: "required argument is missing",
          },
        };
      }
      // Optional and absent -> omit from resolved
      continue;
    }

    if (rawVal === null) {
      return {
        ok: false,
        error: {
          argName: decl.name,
          reason:
            "explicit JSON null is not allowed (use absent for optional arguments)",
        },
      };
    }

    switch (decl.type) {
      case "string": {
        if (typeof rawVal !== "string") {
          return {
            ok: false,
            error: {
              argName: decl.name,
              reason: `expected string, got: ${typeof rawVal}`,
            },
          };
        }
        resolved[decl.name] = { type: "string", strVal: rawVal };
        break;
      }
      case "int64": {
        if (typeof rawVal !== "number" || !Number.isInteger(rawVal)) {
          return {
            ok: false,
            error: {
              argName: decl.name,
              reason: `expected int64, got: ${typeof rawVal}`,
            },
          };
        }
        resolved[decl.name] = { type: "int64", intVal: rawVal };
        break;
      }
      case "bool": {
        if (typeof rawVal !== "boolean") {
          return {
            ok: false,
            error: {
              argName: decl.name,
              reason: `expected bool, got: ${typeof rawVal}`,
            },
          };
        }
        resolved[decl.name] = { type: "bool", boolVal: rawVal };
        break;
      }
      default: {
        return {
          ok: false,
          error: {
            argName: decl.name,
            reason: `internal: unrecognized type`,
          },
        };
      }
    }
  }

  return { ok: true, resolved };
}
