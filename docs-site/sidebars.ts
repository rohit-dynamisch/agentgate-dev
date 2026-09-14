import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'category',
      label: 'Getting Started',
      collapsible: false,
      items: [
        'getting-started/overview',
        'getting-started/prerequisites',
        'getting-started/local-setup',
        'getting-started/first-development-tasks',
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/system-overview',
        'architecture/components',
        'architecture/decision-core',
        'architecture/identity-and-tools',
        'architecture/argument-authorization',
        'architecture/policy-lifecycle',
        'architecture/governance-api',
        'architecture/governance-decision-integration',
        'architecture/gateway-integration',
        'architecture/frontend',
        'architecture/security-model',
      ],
    },
    {
      type: 'category',
      label: 'Repository Guide',
      items: [
        'repository-guide/structure',
        'repository-guide/canonical-docs',
      ],
    },
    {
      type: 'category',
      label: 'Development',
      items: [
        'development/workflow',
        'development/current-status',
        'development/testing',
        'development/deployment-environments',
        'development/ci',
        'development/open-decisions',
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      items: [
        'reference/configuration',
        'reference/api-reference',
        'reference/wire-contract',
        'reference/policy-fixture',
      ],
    },
    {
      type: 'category',
      label: 'History',
      items: [
        'history/evolution',
        'history/commit-index',
      ],
    },
  ],
};

export default sidebars;