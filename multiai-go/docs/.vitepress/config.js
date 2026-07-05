export default {
  lang: 'fr-FR',
  title: 'multiai',
  description: 'Routeur multi-IA — Claude Code, Codex CLI, OpenCode',
  head: [['link', { rel: 'icon', href: '/logo.svg' }]],
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Référence', link: '/reference/commands' },
      { text: 'Avancé', link: '/advanced/custom-profiles' },
      { text: 'GitHub', link: 'https://github.com/lrochetta/multiai' }
    ],
    sidebar: {
      '/guide/': [
        { text: 'Guide', items: [
          { text: 'Premiers pas', link: '/guide/getting-started' },
          { text: 'Installation', link: '/guide/installation' },
          { text: 'Configuration', link: '/guide/configuration' },
          { text: 'Profils', link: '/guide/profiles' },
          { text: 'Dépannage', link: '/guide/troubleshooting' }
        ]}
      ],
      '/reference/': [
        { text: 'Référence', items: [
          { text: 'Commandes', link: '/reference/commands' },
          { text: "Variables d'environnement", link: '/reference/env-variables' },
          { text: 'Codes de sortie', link: '/reference/exit-codes' }
        ]}
      ],
      '/advanced/': [
        { text: 'Avancé', items: [
          { text: 'Profils personnalisés', link: '/advanced/custom-profiles' },
          { text: 'Profils YAML', link: '/advanced/yaml-profiles' },
          { text: 'Profils par projet', link: '/advanced/project-profiles' },
          { text: 'Plugin Hooks', link: '/advanced/plugin-hooks' }
        ]}
      ]
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/lrochetta/multiai' }
    ],
    footer: {
      message: 'Publié sous licence MIT.',
      copyright: 'Copyright © 2026 Laurent Rochetta — rochetta.fr'
    },
    search: { provider: 'local' }
  }
}
