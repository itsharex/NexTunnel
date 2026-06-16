// VitePress 文档站配置，参考开源项目文档网站组织首页、指南和发布说明。
export default {
  title: 'NexTunnel',
  description: '开源内网穿透、P2P 直连与可视化桌面管理工具',
  base: '/NexTunnel/',
  lang: 'zh-CN',
  lastUpdated: true,
  cleanUrls: true,
  head: [
    ['meta', { name: 'theme-color', content: '#00ffff' }],
  ],
  themeConfig: {
    logo: '/logo.png',
    nav: [
      { text: '首页', link: '/' },
      { text: '指南', link: '/guide/getting-started' },
      { text: '桌面端', link: '/desktop/overview' },
      { text: 'CLI', link: '/cli/overview' },
      { text: '部署', link: '/deploy/server' },
      {
        text: 'v0.3.3-alpha',
        items: [
          { text: '更新日志', link: '/changelog' },
          { text: 'GitHub', link: 'https://github.com/Lee-zg/NexTunnel' },
        ],
      },
    ],
    sidebar: {
      '/guide/': [
        {
          text: '入门',
          items: [
            { text: '快速开始', link: '/guide/getting-started' },
            { text: '架构说明', link: '/guide/architecture' },
          ],
        },
      ],
      '/desktop/': [
        {
          text: '桌面端',
          items: [
            { text: '能力总览', link: '/desktop/overview' },
            { text: '隧道与端口', link: '/desktop/tunnels-and-ports' },
            { text: '设置分类', link: '/desktop/settings' },
          ],
        },
      ],
      '/cli/': [
        {
          text: 'CLI',
          items: [
            { text: '使用指南', link: '/cli/overview' },
          ],
        },
      ],
      '/deploy/': [
        {
          text: '部署',
          items: [
            { text: '服务端部署', link: '/deploy/server' },
            { text: '发布流程', link: '/deploy/release' },
          ],
        },
      ],
    },
    socialLinks: [{ icon: 'github', link: 'https://github.com/Lee-zg/NexTunnel' }],
    footer: {
      message: '基于开源许可证发布',
      copyright: 'Copyright 2026 NexTunnel Contributors',
    },
    search: {
      provider: 'local',
      options: {
        translations: {
          button: {
            buttonText: '搜索文档',
            buttonAriaLabel: '搜索文档',
          },
          modal: {
            noResultsText: '无法找到相关结果',
            resetButtonTitle: '清除查询条件',
            footer: {
              selectText: '选择',
              navigateText: '切换',
              closeText: '关闭',
            },
          },
        },
      },
    },
    docFooter: {
      prev: '上一页',
      next: '下一页',
    },
    outline: {
      label: '页面导航',
      level: [2, 3],
    },
    lastUpdated: {
      text: '最后更新于',
      formatOptions: {
        dateStyle: 'short',
        timeStyle: 'medium',
      },
    },
    returnToTopLabel: '回到顶部',
    sidebarMenuLabel: '菜单',
    darkModeSwitchLabel: '主题',
    lightModeSwitchTitle: '切换到浅色模式',
    darkModeSwitchTitle: '切换到深色模式',
  },
  markdown: {
    lineNumbers: true,
    theme: {
      light: 'github-light',
      dark: 'github-dark',
    },
  },
}
