import { beforeEach, afterEach, describe, expect, it, jest } from '@jest/globals';

class MemoryStorage {
    private values = new Map<string, string>();
    getItem(key: string): string | null { return this.values.get(key) ?? null; }
    setItem(key: string, value: string): void { this.values.set(key, value); }
    removeItem(key: string): void { this.values.delete(key); }
    clear(): void { this.values.clear(); }
}

describe('native session bridge', () => {
    const postMessage = jest.fn();
    const dispatchEvent = jest.fn();
    const bridge = { postMessage, onmessage: null as null | ((event: { data: string }) => void) };
    beforeEach(() => {
        jest.resetModules(); jest.useFakeTimers();
        bridge.onmessage = null;
        Object.assign(globalThis, {
            window: { FinanciallyFreeNative: bridge, setTimeout, clearTimeout, dispatchEvent },
            document: { addEventListener: jest.fn() },
            location: { origin: 'https://example.test', pathname: '/finance/mobile.html' },
            localStorage: new MemoryStorage(), sessionStorage: new MemoryStorage(),
            CustomEvent: class { constructor(public type: string, public options: unknown) {} }
        });
    });
    afterEach(() => { jest.useRealTimers(); });

    it('bootstraps without persisting credentials and synchronizes rotation/logout', async () => {
        const native = await import('../native_app.ts');
        localStorage.setItem('ebk_user_token', 'previous-account-secret');
        const initialized = native.initializeNativeApp();
        expect(postMessage).toHaveBeenCalledWith('{"type":"session"}');
        bridge.onmessage!({ data: JSON.stringify({ type: 'session', token: 'initial', user: { username: 'alice' } }) });
        await initialized;
        expect(native.getNativeSession()?.token).toBe('initial');
        expect(localStorage.getItem('ebk_user_token')).toBeNull();
        expect(localStorage.getItem('ebk_user_info')).toBeNull();
        native.setNativeToken('rotated');
        expect(native.getNativeSession()?.token).toBe('rotated');
        expect(postMessage).toHaveBeenLastCalledWith('{"type":"token","token":"rotated"}');
        bridge.onmessage!({ data: JSON.stringify({ type: 'resume', token: 'native-new', user: { username: 'alice' }, revision: 1 }) });
        expect(native.getNativeSession()?.token).toBe('native-new');
        expect(dispatchEvent).toHaveBeenCalled();
        native.clearNativeSession();
        expect(native.getNativeSession()).toBeNull();
        expect(postMessage).toHaveBeenLastCalledWith('{"type":"logout"}');
    });

    it('clears preferences when the account or server subpath changes', async () => {
        const native = await import('../native_app.ts');
        localStorage.setItem('ebk_native_owner', 'https://example.test/other/mobile.html|alice');
        localStorage.setItem('ebk_user_draft_transaction', 'private draft');
        const initialized = native.initializeNativeApp();
        bridge.onmessage!({ data: JSON.stringify({ type: 'session', token: 'initial', user: { username: 'alice' } }) });
        await initialized;
        expect(localStorage.getItem('ebk_user_draft_transaction')).toBeNull();
    });

    it('leaves ordinary browser storage unchanged', async () => {
        delete window.FinanciallyFreeNative;
        const native = await import('../native_app.ts');
        localStorage.setItem('ebk_user_token', 'browser-token');
        await native.initializeNativeApp();
        expect(native.isNativeApp()).toBe(false);
        expect(localStorage.getItem('ebk_user_token')).toBe('browser-token');
    });

    it('times out instead of silently showing a second login', async () => {
        const native = await import('../native_app.ts');
        const initialized = native.initializeNativeApp();
        const failed = expect(initialized).rejects.toThrow('超时');
        jest.advanceTimersByTime(10000);
        await failed;
    });
});
