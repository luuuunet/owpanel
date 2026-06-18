export type RewriteCategory = 'cms' | 'framework' | 'forum' | 'shop' | 'other'

export interface RewriteTemplate {
  id: string
  category: RewriteCategory
  rules: string
}

/** Nginx 规则片段，写入 server 块（可含 location / rewrite 等） */
export const rewriteTemplates: RewriteTemplate[] = [
  {
    id: 'wordpress',
    category: 'cms',
    rules: `# WordPress 伪静态
location / {
    try_files $uri $uri/ /index.php?$args;
}`,
  },
  {
    id: 'typecho',
    category: 'cms',
    rules: `# Typecho
location / {
    try_files $uri $uri/ /index.php$is_args$args;
}`,
  },
  {
    id: 'zblog',
    category: 'cms',
    rules: `# Z-BlogPHP
location / {
    try_files $uri $uri/ /index.php?$query_string;
}`,
  },
  {
    id: 'drupal',
    category: 'cms',
    rules: `# Drupal
location / {
    try_files $uri /index.php?$query_string;
}`,
  },
  {
    id: 'joomla',
    category: 'cms',
    rules: `# Joomla
location / {
    try_files $uri $uri/ /index.php?$args;
}`,
  },
  {
    id: 'ghost',
    category: 'cms',
    rules: `# Ghost（静态导出 / 反向代理站点请用 proxy_pass）
location / {
    try_files $uri $uri/ /index.html;
}`,
  },
  {
    id: 'hexo',
    category: 'cms',
    rules: `# Hexo / 静态博客
location / {
    try_files $uri $uri/ /index.html;
}`,
  },
  {
    id: 'hugo',
    category: 'cms',
    rules: `# Hugo 静态站点
location / {
    try_files $uri $uri/ /index.html;
}`,
  },
  {
    id: 'laravel',
    category: 'framework',
    rules: `# Laravel
location / {
    try_files $uri $uri/ /index.php?$query_string;
}`,
  },
  {
    id: 'thinkphp6',
    category: 'framework',
    rules: `# ThinkPHP 6.x
location / {
    if (!-e $request_filename) {
        rewrite ^(.*)$ /index.php/$1 last;
    }
}`,
  },
  {
    id: 'thinkphp5',
    category: 'framework',
    rules: `# ThinkPHP 5.x
location / {
    if (!-e $request_filename) {
        rewrite ^(.*)$ /index.php?s=/$1 last;
        break;
    }
}`,
  },
  {
    id: 'thinkphp3',
    category: 'framework',
    rules: `# ThinkPHP 3.x
rewrite ^/([0-9]+)/?$ /index.php?p=$1 last;
if (!-e $request_filename) {
    rewrite ^(.*)$ /index.php?s=$1 last;
}`,
  },
  {
    id: 'yii2',
    category: 'framework',
    rules: `# Yii 2
location / {
    try_files $uri $uri/ /index.php?$args;
}`,
  },
  {
    id: 'codeigniter',
    category: 'framework',
    rules: `# CodeIgniter 4
location / {
    try_files $uri $uri/ /index.php$is_args$args;
}`,
  },
  {
    id: 'codeigniter3',
    category: 'framework',
    rules: `# CodeIgniter 3
rewrite ^(.*)$ /index.php/$1 last;`,
  },
  {
    id: 'symfony',
    category: 'framework',
    rules: `# Symfony
location / {
    try_files $uri /index.php$is_args$args;
}`,
  },
  {
    id: 'slim',
    category: 'framework',
    rules: `# Slim Framework
location / {
    try_files $uri $uri/ /index.php?$query_string;
}`,
  },
  {
    id: 'discuz',
    category: 'forum',
    rules: `# Discuz! X（常用伪静态）
rewrite ^([^\\.]*)/topic-(.+)\\.html$ /$1/portal.php?mod=topic&topic=$2 last;
rewrite ^([^\\.]*)/forum-(\\w+)-([0-9]+)\\.html$ /$1/forum.php?mod=forumdisplay&fid=$2&page=$3 last;
rewrite ^([^\\.]*)/thread-(\\w+)-([0-9]+)-([0-9]+)\\.html$ /$1/forum.php?mod=viewthread&tid=$3&page=$4 last;
rewrite ^([^\\.]*)/group-([0-9]+)-([0-9]+)\\.html$ /$1/forum.php?mod=group&fid=$2&page=$3 last;
rewrite ^([^\\.]*)/space-(username)-([0-9]+)\\.html$ /$1/home.php?mod=space&username=$2&page=$3 last;`,
  },
  {
    id: 'phpbb',
    category: 'forum',
    rules: `# phpBB
location / {
    try_files $uri $uri/ /app.php?$query_string;
}`,
  },
  {
    id: 'flarum',
    category: 'forum',
    rules: `# Flarum
location / {
    try_files $uri $uri/ /index.php?$query_string;
}`,
  },
  {
    id: 'ecshop',
    category: 'shop',
    rules: `# ECShop
location / {
    if (!-e $request_filename) {
        rewrite ^/(.*)$ /index.php last;
    }
}`,
  },
  {
    id: 'magento',
    category: 'shop',
    rules: `# Magento 2
location / {
    try_files $uri $uri/ /index.php?$args;
}`,
  },
  {
    id: 'woocommerce',
    category: 'shop',
    rules: `# WooCommerce（基于 WordPress）
location / {
    try_files $uri $uri/ /index.php?$args;
}`,
  },
  {
    id: 'opencart',
    category: 'shop',
    rules: `# OpenCart
location / {
    try_files $uri $uri/ /index.php?$query_string;
}`,
  },
  {
    id: 'dedecms',
    category: 'cms',
    rules: `# DedeCMS（织梦）
rewrite ^/plus/list-(\\d+)-(\\d+)\\.html$ /plus/list.php?tid=$1&PageNo=$2 last;
rewrite ^/plus/view-(\\d+)-(\\d+)\\.html$ /plus/view.php?aid=$1&PageNo=$2 last;
rewrite ^/plus/search\\.html$ /plus/search.php last;
rewrite ^/article/(\\d+)\\.html$ /plus/view.php?aid=$1 last;`,
  },
  {
    id: 'empirecms',
    category: 'cms',
    rules: `# 帝国CMS (EmpireCMS)
rewrite ^/listinfo-(\\d+)-(\\d+)-(\\d+)\\.html$ /e/action/ListInfo.php?classid=$1&page=$2 last;
rewrite ^/infoclass/(\\d+)\\.html$ /e/action/ListInfo.php?classid=$1 last;
rewrite ^/viewinfo-(\\d+)-(\\d+)\\.html$ /e/action/ShowInfo.php?classid=$1&id=$2 last;`,
  },
  {
    id: 'maccms',
    category: 'cms',
    rules: `# 苹果CMS (MacCMS)
location / {
    if (!-e $request_filename) {
        rewrite ^/index.php(.*)$ /index.php?s=$1 last;
        rewrite ^/admin.php(.*)$ /admin.php?s=$1 last;
        rewrite ^/api.php(.*)$ /api.php?s=$1 last;
        rewrite ^(.*)$ /index.php?s=$1 last;
        break;
    }
}`,
  },
  {
    id: 'nextcloud',
    category: 'other',
    rules: `# Nextcloud
location / {
    rewrite ^ /index.php;
}`,
  },
  {
    id: 'mediawiki',
    category: 'other',
    rules: `# MediaWiki
location / {
    try_files $uri $uri/ @rewrite;
}
location @rewrite {
    rewrite ^/(.*)$ /index.php?title=$1&$args last;
}`,
  },
  {
    id: 'spa',
    category: 'other',
    rules: `# Vue / React SPA 单页应用
location / {
    try_files $uri $uri/ /index.html;
}`,
  },
  {
    id: 'phpmyadmin',
    category: 'other',
    rules: `# phpMyAdmin（子目录部署时限制访问）
location ~ ^/phpmyadmin/(libraries|setup/frames|setup/libs) {
    deny all;
}
location /phpmyadmin/ {
    try_files $uri $uri/ /phpmyadmin/index.php?$args;
}`,
  },
  {
    id: 'custom_php',
    category: 'other',
    rules: `# 通用 PHP 前台（单入口 index.php）
location / {
    try_files $uri $uri/ /index.php?$query_string;
}`,
  },
]

export const rewriteCategoryOrder: RewriteCategory[] = ['cms', 'framework', 'forum', 'shop', 'other']

export function getRewriteTemplate(id: string): RewriteTemplate | undefined {
  return rewriteTemplates.find((t) => t.id === id)
}
