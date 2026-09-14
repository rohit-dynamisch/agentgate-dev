import type {ReactNode} from 'react';
import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';

const pillars: {title: string; to: string; body: ReactNode}[] = [
  {
    title: 'Understanding AgentGate',
    to: '/docs/getting-started/overview',
    body: (
      <>
        What the product is, why it exists, and what it currently does — an
        on-boarding path for new engineers.
      </>
    ),
  },
  {
    title: 'How the system works',
    to: '/docs/architecture/system-overview',
    body: (
      <>
        The end-to-end request path, the Go decision core, the policy lifecycle,
        the governance API and the frontend contract layer.
      </>
    ),
  },
  {
    title: 'Developing on this repo',
    to: '/docs/development/workflow',
    body: (
      <>
        The checkpoint-based development model, the AI-assisted workflow, how to
        run tests, and how to reproduce the deployment environments.
      </>
    ),
  },
];

export default function Home(): ReactNode {
  return (
    <Layout
      title="AgentGate — Developer Documentation"
      description="Developer-facing documentation for the AgentGate monorepo: an authorization and governance layer for AI agent tool calls."
    >
      <main
        style={{
          maxWidth: '48rem',
          margin: '0 auto',
          padding: '4rem 1rem',
        }}
      >
        <h1>AgentGate</h1>
        <p className="hero__subtitle">
          A checkpoint in the only path between an AI agent and its tools.
        </p>
        <p>
          This site is the developer knowledge base for the{' '}
          <code>agentgate-dev</code> monorepo. It describes the current
          implementation, explains how the pieces fit together, and tells you
          how to build, run, test, and debug the system — so you do not need a
          knowledge-transfer session to get productive.
        </p>
        <p>
          This is an <strong>early-development</strong> codebase. The durable
          project context lives in the repository's own <code>docs/</code>{' '}
          tree (see the{' '}
          <Link to="/docs/repository-guide/canonical-docs">
            canonical documentation map
          </Link>
          ) — this site is a synthesized, source-checked companion, not a
          replacement.
        </p>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(14rem, 1fr))',
            gap: '1rem',
            marginTop: '2rem',
          }}
        >
          {pillars.map((p) => (
            <div key={p.title} className="card" style={{padding: '0.25rem'}}>
              <div className="card__body">
                <h3>
                  <Link to={p.to}>{p.title}</Link>
                </h3>
                <p style={{margin: 0}}>{p.body}</p>
              </div>
            </div>
          ))}
        </div>
      </main>
    </Layout>
  );
}