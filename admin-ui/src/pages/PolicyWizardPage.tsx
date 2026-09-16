import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAppContext } from "../context/AppContext";
import { usePolicyLifecycle } from "../hooks/usePolicyLifecycle";
import { g3Fixtures, toDryRunComparisonView, toValidationDisplayView } from "../lib/contract";
import { Badge } from "../components/Badge";

type Step = 1 | 2 | 3 | 4;

const STEPS: { n: Step; label: string }[] = [
  { n: 1, label: "Write" },
  { n: 2, label: "Validate" },
  { n: 3, label: "Dry Run" },
  { n: 4, label: "Review & Activate" },
];

function newSample() {
  return { principal_id: "agent-1", principal_roles: "reader", backend_id: "crm", tool_name: "read_record", risk: "read" };
}

export function PolicyWizardPage() {
  const { client, workspaceId } = useAppContext();
  const { state, run } = usePolicyLifecycle(client, workspaceId);
  const navigate = useNavigate();

  const [step, setStep] = useState<Step>(1);
  const [name, setName] = useState("Customer Data Access");
  const [description, setDescription] = useState("Controls access to customer data tools.");
  const [content, setContent] = useState(g3Fixtures.validPolicy1);
  const [candidateVersion, setCandidateVersion] = useState<string | null>(null);
  const [samples, setSamples] = useState([newSample()]);
  const [note, setNote] = useState("");

  const validation = state.validation ? toValidationDisplayView(state.validation) : null;
  const canProceedFromValidate = validation?.tone === "valid" && !!candidateVersion;

  async function handleValidate() {
    const v = await run((s) => s.validateDraft(content));
    if (v?.valid) {
      const rec = await run((s) => s.submitCandidate(content, description || name));
      if (rec) setCandidateVersion(rec.version);
    }
  }

  function updateSample(idx: number, field: keyof ReturnType<typeof newSample>, value: string) {
    setSamples((prev) => prev.map((s, i) => (i === idx ? { ...s, [field]: value } : s)));
  }

  async function handleActivate() {
    if (!candidateVersion) return;
    const resp = await run((s) => s.activate(candidateVersion));
    if (resp) navigate("/policies");
  }

  return (
    <div className="stack">
      <div className="page-header">
        <div>
          <p className="breadcrumb">Policies / Create New Policy</p>
          <h1>Create Candidate Policy</h1>
          <p className="muted">Write and validate your Cedar policy.</p>
        </div>
      </div>

      <div className="stepper">
        {STEPS.map((s) => (
          <button key={s.n} className={`stepper-item ${step === s.n ? "stepper-item--active" : ""} ${step > s.n ? "stepper-item--done" : ""}`} onClick={() => s.n < step && setStep(s.n)}>
            <span className="stepper-index">{s.n}</span>
            {s.label}
          </button>
        ))}
      </div>

      {step === 1 && (
        <section className="card">
          <h2>Policy details</h2>
          <label className="field">
            <span>Name</span>
            <input value={name} onChange={(e) => setName(e.target.value)} />
          </label>
          <label className="field">
            <span>Description</span>
            <input value={description} onChange={(e) => setDescription(e.target.value)} />
          </label>
          <h2>Cedar policy</h2>
          <p className="muted small">
            Raw Cedar text — matches the current API (`createCandidate(content)`). A structured rule builder that
            compiles to Cedar is the documented long-term direction (PROJECT_DEFINITION.md §6) but isn't built on
            either side yet, so it isn't faked here — see admin-ui/FLOW_AND_ARCHITECTURE.md §3 row 3.
          </p>
          <div className="row">
            <button className="button-secondary" onClick={() => setContent(g3Fixtures.validPolicy1)}>
              Template: permit all
            </button>
            <button className="button-secondary" onClick={() => setContent(g3Fixtures.validPolicy2)}>
              Template: forbid all
            </button>
          </div>
          <textarea rows={10} className="code-editor" value={content} onChange={(e) => setContent(e.target.value)} spellCheck={false} />
          <div className="wizard-actions">
            <button className="button-primary" onClick={() => setStep(2)}>
              Next: Validate →
            </button>
          </div>
        </section>
      )}

      {step === 2 && (
        <section className="card">
          <h2>Validation results</h2>
          <button className="button-primary" onClick={handleValidate}>
            Run validation
          </button>
          {validation && (
            <div className={`callout callout--${validation.tone === "valid" ? "allow" : "deny"}`}>
              <strong>{validation.headline}</strong>
              {validation.versionLabel && <p className="mono">candidate version: {validation.versionLabel}</p>}
              {validation.errorMessages?.map((m, i) => (
                <p key={i}>{m}</p>
              ))}
            </div>
          )}
          <div className="wizard-actions">
            <button className="button-secondary" onClick={() => setStep(1)}>
              ← Back
            </button>
            <button className="button-primary" disabled={!canProceedFromValidate} onClick={() => setStep(3)}>
              Next: Dry Run →
            </button>
          </div>
        </section>
      )}

      {step === 3 && (
        <section className="card">
          <h2>Dry-run impact analysis</h2>
          <p className="muted">Comparison against the current active policy. Add sample requests to see how this candidate would change outcomes.</p>
          {samples.map((s, idx) => (
            <div className="row" key={idx}>
              <input placeholder="principal_id" value={s.principal_id} onChange={(e) => updateSample(idx, "principal_id", e.target.value)} />
              <input placeholder="roles" value={s.principal_roles} onChange={(e) => updateSample(idx, "principal_roles", e.target.value)} />
              <input placeholder="backend_id" value={s.backend_id} onChange={(e) => updateSample(idx, "backend_id", e.target.value)} />
              <input placeholder="tool_name" value={s.tool_name} onChange={(e) => updateSample(idx, "tool_name", e.target.value)} />
              <input placeholder="risk" value={s.risk} onChange={(e) => updateSample(idx, "risk", e.target.value)} />
              {samples.length > 1 && <button className="button-secondary" onClick={() => setSamples((prev) => prev.filter((_, i) => i !== idx))}>✕</button>}
            </div>
          ))}
          <div className="row">
            <button className="button-secondary" onClick={() => setSamples((prev) => [...prev, newSample()])}>
              Add sample
            </button>
            <button
              className="button-primary"
              onClick={() =>
                candidateVersion &&
                run((s) =>
                  s.dryRunCompare(
                    candidateVersion,
                    samples.map((sample) => ({
                      principal_id: sample.principal_id,
                      principal_roles: sample.principal_roles.split(",").map((r) => r.trim()).filter(Boolean),
                      backend_id: sample.backend_id,
                      tool_name: sample.tool_name,
                      risk: sample.risk,
                    }))
                  )
                )
              }
            >
              Run Dry Run
            </button>
          </div>

          {state.dryRunResult && (
            <>
              <div className="stat-grid stat-grid--compact">
                <div className="mini-stat">
                  <span className="mini-stat-value">{state.dryRunResult.results.length}</span>
                  <span className="mini-stat-label">Samples evaluated</span>
                </div>
                <div className="mini-stat">
                  <span className="mini-stat-value">{state.dryRunResult.results.filter((r) => r.changed).length}</span>
                  <span className="mini-stat-label">Would change</span>
                </div>
                <div className="mini-stat">
                  <span className="mini-stat-value">{state.dryRunResult.results.filter((r) => !r.changed).length}</span>
                  <span className="mini-stat-label">Unchanged</span>
                </div>
              </div>
              <div className="stack">
                {state.dryRunResult.results.map((r, i) => {
                  const v = toDryRunComparisonView(r);
                  return (
                    <div className={`callout callout--${v.tone === "changed" ? "deny" : "allow"}`} key={i}>
                      <strong>{v.headline}</strong>
                      <p className="muted">before: {v.activeOutcome}</p>
                      <p className="muted">after: {v.candidateOutcome}</p>
                    </div>
                  );
                })}
              </div>
            </>
          )}

          <div className="wizard-actions">
            <button className="button-secondary" onClick={() => setStep(2)}>
              ← Back
            </button>
            <button className="button-primary" onClick={() => setStep(4)}>
              Next: Review & Activate →
            </button>
          </div>
        </section>
      )}

      {step === 4 && candidateVersion && (
        <section className="card">
          <h2>Activate policy</h2>
          <div className="callout callout--stale">
            <strong>This will replace the current active policy.</strong>
            <p>Review the dry-run results above before confirming.</p>
          </div>
          <div className="review-grid">
            <div>
              <span className="muted">Name</span>
              <p>{name}</p>
            </div>
            <div>
              <span className="muted">Candidate version</span>
              <p className="mono">{candidateVersion.slice(0, 20)}…</p>
            </div>
          </div>
          <label className="field">
            <span>Reason for activation</span>
            <input value={note} onChange={(e) => setNote(e.target.value)} placeholder="e.g. approved for production" />
          </label>
          <p className="muted small">Local note only — the current activation API doesn't record a reason field.</p>
          <div className="wizard-actions">
            <button className="button-secondary" onClick={() => setStep(3)}>
              ← Back
            </button>
            <button className="button-danger" onClick={handleActivate}>
              Activate Policy
            </button>
          </div>
          {state.activationStatus === "error" && <Badge tone="error">{state.lastError}</Badge>}
        </section>
      )}
    </div>
  );
}
