import { isEnableDebug } from './settings.ts';

function logDebug(msg: string, obj?: unknown): void {
    if (isEnableDebug()) {
        if (obj) {
            console.debug('[financially_free Debug] ' + msg, obj);
        } else {
            console.debug('[financially_free Debug] ' + msg);
        }
    }
}

function logInfo(msg: string, obj?: unknown): void {
    if (obj) {
        console.info('[financially_free Info] ' + msg, obj);
    } else {
        console.info('[financially_free Info] ' + msg);
    }
}

function logWarn(msg: string, obj?: unknown): void {
    if (obj) {
        console.warn('[financially_free Warn] ' + msg, obj);
    } else {
        console.warn('[financially_free Warn] ' + msg);
    }
}

function logError(msg: string, obj?: unknown): void {
    if (obj) {
        console.error('[financially_free Error] ' + msg, obj);
    } else {
        console.error('[financially_free Error] ' + msg);
    }
}

export default {
    debug: logDebug,
    info: logInfo,
    warn: logWarn,
    error: logError
};
