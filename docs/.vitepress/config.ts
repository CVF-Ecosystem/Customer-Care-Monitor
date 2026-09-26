import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Customer Care Monitor AI',
  description: 'Giám sát chất lượng chăm sóc khách hàng cho một workspace',
  lang: 'vi-VN',
  base: '/Customer-Care-Monitor-AI/',
  rewrites: {
    'home.md': 'index.md',
  },
  srcExclude: [
    'INDEX.md',
    'CVF_BOOTSTRAP_LOG_*.md',
    'catalog/**',
    'decisions/**',
    'roadmaps/**',
    'specs/**',
    'work_orders/**',
    'reviews/CVF_*.md',
    'reviews/CONTRIBUTOR_*.md',
    'reviews/README.md',
  ],

  head: [
    ['meta', { name: 'theme-color', content: '#1976D2' }],
  ],

  themeConfig: {
    nav: [
      { text: 'Bắt đầu', link: '/guide/installation' },
      { text: 'Khác biệt & trạng thái', link: '/guide/introduction' },
      { text: 'Rà soát tài liệu', link: '/reviews/USER_DOCS_AUDIT_2026-09-26' },
    ],

    sidebar: [
      {
        text: 'Bắt đầu',
        items: [
          { text: 'Giới thiệu', link: '/guide/introduction' },
          { text: 'Cài đặt', link: '/guide/installation' },
          { text: 'Cập nhật phiên bản', link: '/guide/updates' },
          { text: 'Tên miền & SSL', link: '/guide/domain-ssl' },
          { text: 'Thiết lập ban đầu', link: '/guide/initial-setup' },
        ],
      },
      {
        text: 'Sử dụng',
        items: [
          { text: 'Cấu hình chung', link: '/usage/general-settings' },
          { text: 'Cấu hình AI', link: '/usage/ai-settings' },
          { text: 'Kết nối Zalo OA', link: '/usage/channels' },
          { text: 'Kết nối Facebook', link: '/usage/facebook' },
          { text: 'Kết nối Pancake', link: '/usage/pancake' },
          { text: 'Quản lý tin nhắn', link: '/usage/messages' },
          { text: 'Tạo công việc', link: '/usage/jobs' },
          { text: 'Xem kết quả', link: '/usage/results' },
          { text: 'Thông báo', link: '/usage/notifications' },
          { text: 'Dashboard', link: '/usage/dashboard' },
          { text: 'Nhật ký hệ thống', link: '/usage/activity-logs' },
          { text: 'Chi phí AI', link: '/usage/cost-logs' },
        ],
      },
      {
        text: 'Quản trị',
        items: [
          { text: 'Người dùng & phân quyền', link: '/admin/users' },
          { text: 'Phạm vi một workspace', link: '/admin/multi-tenant' },
        ],
      },
      {
        text: 'Tham khảo',
        items: [
          { text: 'Biến môi trường', link: '/reference/env-vars' },
          { text: 'Rà soát tài liệu', link: '/reviews/USER_DOCS_AUDIT_2026-09-26' },
        ],
      },
      {
        text: 'Hỗ trợ',
        items: [
          { text: 'Định hướng sản phẩm', link: '/PRODUCT_DIRECTION' },
        ],
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/CVF-Ecosystem/Customer-Care-Monitor-AI' },
    ],

    search: {
      provider: 'local',
    },

    footer: {
      message: 'Phát hành theo giấy phép MIT',
      copyright: 'Copyright 2026 CVF-Ecosystem; nguồn CQA © SePay',
    },
  },
})
