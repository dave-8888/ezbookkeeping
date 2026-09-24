import { describe, expect, test } from '@jest/globals';

import { getSocialGiftStatus, isSocialGiftEventNameValid, resolveValidSocialGiftSelection } from '@/lib/social_gift.ts';

describe('social gift balance status', () => {
    test.each([
        [100, 'receivable'],
        [-1, 'payable'],
        [0, 'settled']
    ])('maps net amount %s to %s', (amount, expected) => {
        expect(getSocialGiftStatus(amount as number)).toBe(expected);
    });
});

describe('social gift editor validation and remembered selection', () => {
    test('requires a name only for other events', () => {
        expect(isSocialGiftEventNameValid('other', '  ')).toBe(false);
        expect(isSocialGiftEventNameValid('other', 'Custom event')).toBe(true);
        expect(isSocialGiftEventNameValid('wedding', '')).toBe(true);
    });

    test('rejects sentinel and removed default accounts while retaining an empty category choice', () => {
        expect(resolveValidSocialGiftSelection(undefined, ['12'], '0').value).toBe('12');
        expect(resolveValidSocialGiftSelection(undefined, [], '0').value).toBe('');
        expect(resolveValidSocialGiftSelection(undefined, ['12']).value).toBe('');
    });

    test('uses valid remembered values and falls back when they disappear', () => {
        expect(resolveValidSocialGiftSelection('2', ['1', '2'], '1')).toEqual({ value: '2', preferenceValid: true });
        expect(resolveValidSocialGiftSelection('3', ['1', '2'], '1')).toEqual({ value: '1', preferenceValid: false });
    });
});
