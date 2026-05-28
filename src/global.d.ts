declare const __FINANCIALLY_FREE_IS_PRODUCTION__: boolean;
declare const __FINANCIALLY_FREE_VERSION__: string;
declare const __FINANCIALLY_FREE_BUILD_UNIX_TIME__: string;
declare const __FINANCIALLY_FREE_BUILD_COMMIT_HASH__: string;
declare const __FINANCIALLY_FREE_LICENSE__: string;
declare const __FINANCIALLY_FREE_CONTRIBUTORS__: ContributorInfo;
declare const __FINANCIALLY_FREE_THIRD_PARTY_LICENSES__: LicenseInfo[];

declare interface ContributorInfo {
    code: string[];
    translators: Record<string, string[]>;
}

declare interface LicenseInfo {
    name: string;
    copyright?: string;
    url?: string;
    licenseUrl?: string;
}

interface Window {
    FINANCIALLY_FREE_SERVER_SETTINGS?: {
        [key: string]: string | number | boolean | undefined | null;
    };
}

interface Navigator {
    browserLanguage?: string;
}

interface Credential {
    rawId: ArrayBuffer;
    response: {
        clientDataJSON: ArrayBuffer;
        attestationObject: ArrayBuffer;
        userHandle: ArrayBuffer;
    };
}
