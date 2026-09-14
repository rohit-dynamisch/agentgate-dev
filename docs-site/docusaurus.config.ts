import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const REPO_OWNER = 'rushi-dynmsh';
const REPO_NAME = 'agentgate-dev';
const REPO_URL = `https://github.com/${REPO_OWNER}/${REPO_NAME}`;

const config: Config = {
  title: 'AgentGate',
  tagline: 'A checkpoint in the only path between an AI agent and its tools',
  // Deployment target is not fixed yet. These values are GitHub-Pages-shaped
  // placeholders; update `url`/`baseUrl` when the site gets a real host.
  url: 'https://rushi-dynmsh.github.io',
  baseUrl: '/',

  // Future flags, see https://docusaurus.io/docs/api/docusaurus-config#future
  // Deliberately NOT enabling `future.v4` / the Faster (Rspack) bundler: the
  // Rspack native binding pulled in by @docusaurus/faster is broken on this
  // Windows toolchain and is a native-binary CI risk. Standard webpack bundler
  // is used instead.
  future: {},

  onBrokenLinks: 'throw',

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang.
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  // Mermaid diagrams are used across the architecture and development docs.
  markdown: {
    mermaid: true,
  },
  themes: ['@docusaurus/theme-mermaid'],

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: `${REPO_URL}/tree/main/docs-site/`,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'AgentGate',
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Docs',
        },
        {
          href: REPO_URL,
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {
              label: 'Overview',
              to: '/docs/getting-started/overview',
            },
            {
              label: 'System Architecture',
              to: '/docs/architecture/system-overview',
            },
            {
              label: 'Current Status',
              to: '/docs/development/current-status',
            },
          ],
        },
        {
          title: 'Project',
          items: [
            {
              label: 'Canonical docs (repo `docs/`)',
              href: `${REPO_URL}/tree/main/docs`,
            },
            {
              label: 'Project Definition',
              href: `${REPO_URL}/blob/main/docs/PROJECT_DEFINITION.md`,
            },
            {
              label: 'GitHub Repository',
              href: REPO_URL,
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} AgentGate. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
    mermaid: {
      theme: {
        light: 'neutral',
        dark: 'dark',
      },
    },
  } satisfies Preset.ThemeConfig,
};

export default config;