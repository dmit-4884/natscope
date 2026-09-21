import { defineConfig } from 'vitepress'

const SITE_URL = 'https://natscope.app/'
const DESCRIPTION = 'Web UI for NATS JetStream — browse, decode, publish and manage streams.'

function pageUrl(relativePath) {
  const path = relativePath.replace(/index\.md$/, '').replace(/\.md$/, '.html')
  return SITE_URL + path
}

export default defineConfig({
  title: 'Natscope',
  description: DESCRIPTION,
  base: '/',
  lang: 'en-US',
  cleanUrls: false,
  lastUpdated: true,
  sitemap: {
    hostname: SITE_URL
  },
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/favicon-32.png' }],
    ['link', { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'Natscope' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }]
  ],
  transformHead({ pageData }) {
    const url = pageUrl(pageData.relativePath)
    const title = pageData.frontmatter.title || pageData.title || 'Natscope'
    const description = pageData.frontmatter.description || pageData.description || DESCRIPTION

    return [
      ['link', { rel: 'canonical', href: url }],
      ['meta', { property: 'og:url', content: url }],
      ['meta', { property: 'og:title', content: title }],
      ['meta', { property: 'og:description', content: description }]
    ]
  },
  themeConfig: {
    logo: '/logo.svg',
    search: {
      provider: 'local'
    },
    nav: [
      { text: 'Docs', link: '/guide/what-is-natscope', activeMatch: '/(guide|reference|help)/' },
      { text: 'Releases', link: 'https://github.com/dmit-4884/natscope/releases' }
    ],
    sidebar: [
      {
        text: 'Getting Started',
        items: [
          { text: 'What is Natscope', link: '/guide/what-is-natscope' },
          { text: 'Installation', link: '/guide/installation' },
          { text: 'Quick start', link: '/guide/quick-start' }
        ]
      },
      {
        text: 'Guide',
        items: [
          { text: 'Connections', link: '/guide/connections' },
          { text: 'Streams', link: '/guide/streams' },
          { text: 'Messages', link: '/guide/messages' },
          { text: 'Live tail', link: '/guide/live-tail' },
          { text: 'Consumers', link: '/guide/consumers' },
          { text: 'Publishing', link: '/guide/publish' },
          { text: 'Protobuf', link: '/guide/protobuf' },
          { text: 'Key/Value', link: '/guide/kv' },
          { text: 'Object Store', link: '/guide/object-store' },
          { text: 'Templates', link: '/guide/templates' },
          { text: 'Publish history', link: '/guide/history' },
          { text: 'Workspace', link: '/guide/workspace' },
          { text: 'Settings', link: '/guide/settings' }
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'Configuration', link: '/reference/configuration' },
          { text: 'Remote access', link: '/reference/remote-access' },
          { text: 'Secrets', link: '/reference/secrets' },
          { text: 'Data locations', link: '/reference/data-locations' }
        ]
      },
      {
        text: 'Help',
        items: [{ text: 'Troubleshooting', link: '/help/troubleshooting' }]
      }
    ],
    socialLinks: [{ icon: 'github', link: 'https://github.com/dmit-4884/natscope' }],
    editLink: {
      pattern: 'https://github.com/dmit-4884/natscope/edit/main/site/:path',
      text: 'Edit this page on GitHub'
    },
    footer: {
      message: 'Released under the Apache License 2.0.',
      copyright: 'Copyright © 2026 The Natscope Authors'
    },
    outline: [2, 3]
  }
})
