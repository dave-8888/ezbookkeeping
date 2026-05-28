import fs from 'fs';
import { resolve } from 'path';

import { type UserConfig, type Plugin, defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue';
import vuetify from 'vite-plugin-vuetify';
import { VitePWA } from 'vite-plugin-pwa';
import Checker from 'vite-plugin-checker';
import git from 'git-rev-sync';

import packageFile from './package.json';
import contributorsFile from './contributors.json';
import thirdPartyLicenseFile from './third-party-dependencies.json';

const SRC_DIR = resolve(__dirname, './src');
const PUBLIC_DIR = resolve(__dirname, './public');
const BUILD_DIR = resolve(__dirname, './dist',);

function getPortFromEnv(name: string, fallback: number): number {
    const value = process.env[name];

    if (!value) {
        return fallback;
    }

    const port = Number(value);

    if (!Number.isInteger(port) || port < 1 || port > 65535) {
        return fallback;
    }

    return port;
}

function injectFramework7CssFile({ htmlFileName, placeHolders }: { htmlFileName: string, placeHolders: { name: string, srcFileName: string, distFileNamePrefix: string }[] }): Plugin[] {
    return [
        {
            name: 'inject-framework7-css-file:serve',
            apply: 'serve',
            enforce: 'post',
            transformIndexHtml(html: string): string {
                for (const placeholder of placeHolders) {
                    html = html.replace(`{{${placeholder.name}}}`, `${placeholder.srcFileName}`);
                }
                return html;
            }
        },
        {
            name: 'inject-framework7-css-file:build',
            apply: 'build',
            enforce: 'post',
            generateBundle(_, bundle): void {
                const placeholderCssFilePathMap: Record<string, string> = {};

                for (const fileName of Object.keys(bundle)) {
                    for (const placeholder of placeHolders) {
                        if (fileName.startsWith(placeholder.distFileNamePrefix)) {
                            placeholderCssFilePathMap[placeholder.name] = fileName;
                            break;
                        }
                    }
                }

                const htmlAsset = bundle[htmlFileName];

                if (!htmlAsset || htmlAsset.type !== 'asset') {
                    return;
                }

                let html = htmlAsset.source as string;

                for (const [placeholder, filePath] of Object.entries(placeholderCssFilePathMap)) {
                    html = html.replace(`{{${placeholder}}}`, `./${filePath}`);
                }

                htmlAsset.source = html;
            }
        }
    ];
}

export default defineConfig(() => {
    const licenseContent = fs.readFileSync('./LICENSE', { encoding: 'utf-8' });
    const buildUnixTime = process.env['buildUnixTime'] || '';
    const backendPort = getPortFromEnv('BACKEND_PORT', getPortFromEnv('APP_PORT', 8888));
    const frontendHost = process.env['FRONTEND_HOST'] || '0.0.0.0';
    const frontendPort = getPortFromEnv('FRONTEND_PORT', 8889);
    const backendTarget = `http://127.0.0.1:${backendPort}/`;

    const options: UserConfig = {
        root: SRC_DIR,
        publicDir: PUBLIC_DIR,
        base: './',
        define: {
            __FINANCIALLY_FREE_IS_PRODUCTION__: process.env['NODE_ENV'] === 'production',
            __FINANCIALLY_FREE_VERSION__: JSON.stringify(packageFile.version),
            __FINANCIALLY_FREE_BUILD_UNIX_TIME__: JSON.stringify(buildUnixTime),
            __FINANCIALLY_FREE_BUILD_COMMIT_HASH__: JSON.stringify(git.short()),
            __FINANCIALLY_FREE_CONTRIBUTORS__: JSON.stringify(contributorsFile),
            __FINANCIALLY_FREE_LICENSE__: JSON.stringify(licenseContent),
            __FINANCIALLY_FREE_THIRD_PARTY_LICENSES__: JSON.stringify(thirdPartyLicenseFile)
        },
        plugins: [
            vue({
                template: {
                    compilerOptions: {
                        isCustomElement: tag => tag.includes('swiper-')
                    }
                }
            }),
            vuetify({
                styles: {
                    configFile: 'styles/desktop/configured-variables/_vuetify.scss'
                }
            }),
            injectFramework7CssFile({
                htmlFileName: 'mobile.html',
                placeHolders: [
                    {
                        name: 'framework7-ltr-css-filepath',
                        srcFileName: 'mobile-ltr.scss',
                        distFileNamePrefix: 'css/vendor-framework7-ltr'
                    },
                    {
                        name: 'framework7-rtl-css-filepath',
                        srcFileName: 'mobile-rtl.scss',
                        distFileNamePrefix: 'css/vendor-framework7-rtl'
                    }
                ]
            }),
            Checker({
                vueTsc: true
            }),
            VitePWA({
                filename: 'sw.js',
                manifestFilename: 'manifest.json',
                strategies: 'generateSW',
                injectRegister: false,
                manifest: {
                    name: 'financially_free',
                    short_name: 'financially_free',
                    description: 'A lightweight, self-hosted personal finance app with a user-friendly interface and powerful bookkeeping features.',
                    theme_color: '#C67E48',
                    background_color: '#F6F7F8',
                    start_url: './',
                    scope: './',
                    display: 'standalone',
                    lang: 'zh-CN',
                    related_applications: [],
                    prefer_related_applications: false,
                    icons: [
                        {
                            src: 'img/financially_free-192.png',
                            sizes: '192x192',
                            type: 'image/png'
                        },
                        {
                            src: 'img/financially_free-512.png',
                            sizes: '512x512',
                            type: 'image/png'
                        }
                    ]
                },
                workbox: {
                    globDirectory: 'dist/',
                    globPatterns: ['**/*.{js,css,html,ico,png,jpg,jpeg,gif,tiff,bmp,ttf,woff,woff2,svg,eot}'],
                    globIgnores: [
                        'index.html',
                        'mobile.html',
                        'desktop.html',
                        'robots.txt',
                        'img/desktop/*',
                        'fonts/*.eot',
                        'fonts/*.ttf',
                        'fonts/*.svg',
                        'fonts/*.woff',
                        'css/*.css',
                        'js/*.js'
                    ],
                    runtimeCaching: [
                        {
                            urlPattern: /.*\/(mobile|mobile\/|desktop|desktop\/)$/,
                            handler: 'NetworkFirst'
                        },
                        {
                            urlPattern: /.*\/(mobile|mobile\/)#!\//,
                            handler: 'NetworkFirst'
                        },
                        {
                            urlPattern: /.*\/(desktop|desktop\/)#\//,
                            handler: 'NetworkFirst'
                        },
                        {
                            urlPattern: /.*\/(index\.html|mobile\.html|desktop\.html)/,
                            handler: 'NetworkFirst'
                        },
                        {
                            urlPattern: /.*\/img\/desktop\/.*\.(png|jpg|jpeg|gif|tiff|bmp|svg)/,
                            handler: 'StaleWhileRevalidate'
                        },
                        {
                            urlPattern: /.*\/fonts\/.*\.(eot|ttf|svg|woff)/,
                            handler: 'CacheFirst'
                        },
                        {
                            urlPattern: /.*\/css\/.*\.css/,
                            handler: 'CacheFirst'
                        },
                        {
                            urlPattern: /.*\/js\/.*\.js/,
                            handler: 'CacheFirst'
                        }
                    ],
                    navigateFallback: '',
                    skipWaiting: true,
                    clientsClaim: true
                }
            })
        ],
        css: {
            preprocessorOptions: {
                scss: {
                    quietDeps: true,
                    silenceDeprecations: ['if-function']
                },
                sass: {
                    quietDeps: true,
                    silenceDeprecations: ['if-function']
                }
            }
        },
        build: {
            target: [
                'chrome91',
                'edge91',
                'firefox91',
                'safari15.4'
            ],
            outDir: BUILD_DIR,
            sourcemap: false,
            assetsInlineLimit: 0,
            chunkSizeWarningLimit: 2200,
            emptyOutDir: true,
            rollupOptions: {
                input: {
                    index: resolve(SRC_DIR, 'index.html'),
                    desktop: resolve(SRC_DIR, 'desktop.html'),
                    mobile: resolve(SRC_DIR, 'mobile.html'),
                    'vendor-framework7-ltr': resolve(SRC_DIR, 'mobile-ltr.scss'),
                    'vendor-framework7-rtl': resolve(SRC_DIR, 'mobile-rtl.scss')
                },
                output: {
                    assetFileNames: assetInfo => {
                        const fileExt = assetInfo.names[0]?.split('.')[1];

                        if (!fileExt) {
                            throw new Error('Invalid asset file name.');
                        }

                        let assetType = fileExt;

                        if (/png|jpe?g|gif|tiff|bmp|ico/i.test(fileExt)) {
                            assetType = 'img';
                        } else if (/ttf|woff|woff2|svg|eot/i.test(fileExt)) {
                            assetType = 'fonts';
                        }

                        return `${assetType}/[name]-[hash][extname]`;
                    },
                    chunkFileNames: 'js/[name]-[hash].js',
                    entryFileNames: 'js/[name]-[hash].js',
                    manualChunks: id => {
                        if (/[\\/]node_modules[\\/]leaflet[\\/]/i.test(id)) {
                            return 'leaflet';
                        } else if (/[\\/]node_modules[\\/](moment|moment-timezone)[\\/]/i.test(id)) {
                            return 'moment';
                        } else if (/[\\/]node_modules[\\/](dom7|framework7.*|skeleton-elements|swiper)[\\/]/i.test(id)) {
                            return 'vendor-mobile';
                        } else if (/[\\/]node_modules[\\/](vuetify|vue-router|vue3-perfect-scrollbar|perfect-scrollbar|vuedraggable|sortablejs|@mdi.*)[\\/]/i.test(id)) {
                            return 'vendor-desktop';
                        } else if (/[\\/]node_modules[\\/](echarts|zrender|tslib|resize-detector|vue-echarts)[\\/]/i.test(id)) {
                            return 'vendor-desktop';
                        } else if (/plugin-vuetify:/i.test(id)) {
                            return 'vendor-desktop';
                        } else if (/[\\/]node_modules[\\/]/i.test(id)) {
                            return 'vendor-common';
                        } else if (/[\\/]src[\\/](core|consts|models|stores)[\\/]/i.test(id)) {
                            return 'common';
                        } else if (/[\\/]src[\\/]lib[\\/](map[\\/]|ui[\\/]common|[a-zA-Z0-9-_]+\.(js|ts))/i.test(id)) {
                            return 'common';
                        } else if (/[\\/]src[\\/]components[\\/](base|common)[\\/]/i.test(id)) {
                            return 'common';
                        } else if (/[\\/]src[\\/]views[\\/]base[\\/]/i.test(id)) {
                            return 'common';
                        } else if (/[\\/]src[\\/]locales[\\/](helpers\.(js|ts))/i.test(id)) {
                            return 'common';
                        } else if (/[\\/]src[\\/]locales[\\/]/i.test(id)) {
                            return 'locales';
                        } else {
                            return null;
                        }
                    }
                },
                treeshake: false
            },
        },
        resolve: {
            alias: {
                '@': SRC_DIR,
            },
        },
        server: {
            host: frontendHost,
            port: frontendPort,
            strictPort: true,
            proxy: {
                '/server_settings.js': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/mobile/server_settings.js': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/desktop/server_settings.js': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/oauth2': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/api': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/mcp': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/avatar': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/pictures': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/qrcode': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/proxy': {
                    target: backendTarget,
                    changeOrigin: true
                },
                '/_AMapService': {
                    target: backendTarget,
                    changeOrigin: true
                }
            }
        },
    };

    return options;
})
