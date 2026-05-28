export function getLicense(): string {
    return __FINANCIALLY_FREE_LICENSE__;
}

export function getThirdPartyLicenses(): LicenseInfo[] {
    return __FINANCIALLY_FREE_THIRD_PARTY_LICENSES__ || [];
}
