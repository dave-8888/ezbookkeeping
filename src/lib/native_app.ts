import type { UserBasicInfo } from '@/models/user.ts';

interface NativeSession { token: string; user: UserBasicInfo }
interface NativeBridge {
    postMessage(message: string): void;
    onmessage: ((event: { data: string }) => void) | null;
}
declare global {
    interface Window { FinanciallyFreeNative?: NativeBridge }
}

let session: NativeSession | null = null;
export const isNativeApp = (): boolean => typeof window !== 'undefined' && !!window.FinanciallyFreeNative;
export const getNativeSession = (): NativeSession | null => session;

export function notifyNative(type: string, data: Record<string, unknown> = {}): void {
    window.FinanciallyFreeNative?.postMessage(JSON.stringify({ type, ...data }));
}

export function setNativeToken(token: string): void {
    if (session) session.token = token;
    notifyNative('token', { token });
}

export function setNativeUser(user: UserBasicInfo): void {
    if (session) session.user = user;
    notifyNative('user', { user });
}

export function clearNativeSession(): void {
    session = null;
    notifyNative('logout');
}

// Called before importing the app: stores read the session during their creation.
// Credentials stay in memory; only Android's Keystore-backed store persists them.
export async function initializeNativeApp(): Promise<void> {
    const bridge = window.FinanciallyFreeNative;
    if (!bridge) return;
    await new Promise<void>((resolve, reject) => {
        const timer = window.setTimeout(() => reject(new Error('连接 App 登录状态超时，请返回重试')), 10000);
        bridge.onmessage = event => {
            const message = JSON.parse(event.data);
            if (message.type === 'session' && message.token && message.user?.username) {
                session = { token: message.token, user: message.user };
                window.clearTimeout(timer);
                resolve();
            } else if (message.type === 'resume' && session) {
                session.token = message.token;
                session.user = message.user;
                window.dispatchEvent(new CustomEvent('native-resume', { detail: message }));
            } else if (message.type === 'back') {
                window.dispatchEvent(new Event('native-back'));
            }
        };
        notifyNative('session');
    });

    // Web preferences/caches must never cross accounts or server subpaths.
    const owner = location.origin + location.pathname + '|' + session!.user.username;
    if (localStorage.getItem('ebk_native_owner') !== owner) {
        localStorage.clear();
        sessionStorage.clear();
        localStorage.setItem('ebk_native_owner', owner);
    }
    localStorage.removeItem('ebk_user_token');
    localStorage.removeItem('ebk_user_info');

    // Blob exports use Android's document picker, including links created by Vue.
    document.addEventListener('click', async event => {
        const link = (event.target as Element)?.closest<HTMLAnchorElement>('a[download]');
        if (!link || !link.href.startsWith('blob:')) return;
        event.preventDefault();
        event.stopImmediatePropagation();
        try {
            const blob = await (await fetch(link.href)).blob();
            // Bound the bridge payload; large exports remain available in the browser.
            if (blob.size > 20 * 1024 * 1024) throw new Error('文件超过 20 MB，请在浏览器中导出');
            const reader = new FileReader();
            reader.onload = () => notifyNative('download', {
                name: link.download || 'export.csv', mime: blob.type,
                data: String(reader.result).split(',')[1]
            });
            reader.readAsDataURL(blob);
        } catch (error) {
            notifyNative('error', { message: error instanceof Error ? error.message : '导出失败，请重试' });
        }
    }, true);
}
