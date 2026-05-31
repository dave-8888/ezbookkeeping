import { describe, expect, test } from '@jest/globals';

import { AccountCategory, AccountType } from '@/core/account.ts';
import { Account, type AccountInfoResponse, type CategorizedAccount } from '@/models/account.ts';

import { getAllFilteredAccountsBalance } from '@/lib/account.ts';

function createAccount(account: Partial<AccountInfoResponse> & Pick<AccountInfoResponse, 'id' | 'name'>): AccountInfoResponse {
    return {
        id: account.id,
        name: account.name,
        parentId: account.parentId ?? '0',
        category: account.category ?? AccountCategory.Cash.type,
        type: account.type ?? AccountType.SingleAccount.type,
        icon: account.icon ?? '1',
        color: account.color ?? '1',
        currency: account.currency ?? 'USD',
        balance: account.balance ?? 0,
        comment: account.comment ?? '',
        displayOrder: account.displayOrder ?? 0,
        isAsset: account.isAsset ?? true,
        isLiability: account.isLiability ?? false,
        hidden: account.hidden ?? false,
        subAccounts: account.subAccounts
    };
}

function createCategorizedAccountsMap(accounts: Account[]): Record<number, CategorizedAccount> {
    return {
        [AccountCategory.Cash.type]: {
            category: AccountCategory.Cash.type,
            name: AccountCategory.Cash.name,
            icon: AccountCategory.Cash.defaultAccountIconId,
            accounts
        }
    };
}

describe('getAllFilteredAccountsBalance', () => {
    test('excludes hidden accounts by default and includes them when requested', () => {
        const visibleAccount = Account.of(createAccount({
            id: 'visible_account',
            name: 'Visible Account',
            balance: 100
        }));
        const hiddenAccount = Account.of(createAccount({
            id: 'hidden_account',
            name: 'Hidden Account',
            balance: 200,
            hidden: true
        }));
        const accountWithSubAccounts = Account.of(createAccount({
            id: 'parent_account',
            name: 'Parent Account',
            type: AccountType.MultiSubAccounts.type,
            subAccounts: [
                createAccount({
                    id: 'visible_sub_account',
                    name: 'Visible Sub Account',
                    parentId: 'parent_account',
                    balance: 300
                }),
                createAccount({
                    id: 'hidden_sub_account',
                    name: 'Hidden Sub Account',
                    parentId: 'parent_account',
                    balance: 400,
                    hidden: true
                })
            ]
        }));
        const categorizedAccountsMap = createCategorizedAccountsMap([
            visibleAccount,
            hiddenAccount,
            accountWithSubAccounts
        ]);
        const isAssetAccount = (account: Account) => account.isAsset;

        expect(getAllFilteredAccountsBalance(categorizedAccountsMap, isAssetAccount).map(item => item.balance))
            .toStrictEqual([100, 300]);
        expect(getAllFilteredAccountsBalance(categorizedAccountsMap, isAssetAccount, true).map(item => item.balance))
            .toStrictEqual([100, 200, 300, 400]);
    });
});
