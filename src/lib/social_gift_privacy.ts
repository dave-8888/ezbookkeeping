// Access exists only in memory and is discarded whenever the page is left.
let token = '';
export function setSocialGiftToken(value: string): void { token = value; }
export function getSocialGiftToken(): string { return token; }
export function socialGiftPrivacyHeaders(): { headers: Record<string, string> } {
    return { headers: { 'X-Social-Gift-Token': token } };
}
