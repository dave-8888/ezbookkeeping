import { initializeNativeApp } from '@/lib/native_app.ts';

initializeNativeApp().then(() => import('./mobile-main.ts')).catch(error => {
    const app = document.getElementById('app');
    if (app) app.textContent = error instanceof Error ? error.message : '加载失败，请返回重试';
});
