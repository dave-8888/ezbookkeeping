import type { TransactionInfoResponse } from '@/models/transaction.ts';

export type SocialGiftDirection = 'sent' | 'received';
export type SocialGiftStatus = 'receivable' | 'payable' | 'settled';

export interface SocialGiftImportMetadata {
    readonly contactKey: string;
    readonly contactName: string;
    readonly relationship: string;
    readonly contactNote: string;
    readonly eventType: SocialGiftEventType;
    readonly eventName: string;
}
export type SocialGiftEventType = 'wedding' | 'newborn' | 'birthday' | 'housewarming' | 'education' | 'funeral' | 'other';

export interface SocialGiftContactCreateRequest {
    readonly name: string;
    readonly relationship: string;
    readonly comment: string;
}

export interface SocialGiftContactModifyRequest extends SocialGiftContactCreateRequest {
    readonly id: string;
}

export interface SocialGiftContactDeleteRequest { readonly id: string }

export interface SocialGiftRecordCreateRequest {
    readonly currency: string;
    readonly contactId: string;
    readonly newContact?: SocialGiftContactCreateRequest;
    readonly direction: SocialGiftDirection;
    readonly eventType: SocialGiftEventType;
    readonly eventName: string;
    readonly categoryId: string;
    readonly time: number;
    readonly utcOffset: number;
    readonly accountId: string;
    readonly amount: number;
    readonly comment: string;
    readonly clientSessionId: string;
}

export interface SocialGiftRecordModifyRequest extends Omit<SocialGiftRecordCreateRequest, 'newContact' | 'clientSessionId'> {
    readonly id: string;
}

export interface SocialGiftRecordDeleteRequest { readonly id: string }

export interface SocialGiftBalanceResponse {
    readonly currency: string;
    readonly sentAmount: number;
    readonly receivedAmount: number;
    readonly netAmount: number;
    readonly status: SocialGiftStatus;
}

export interface SocialGiftContactInfoResponse {
    readonly id: string;
    readonly name: string;
    readonly relationship: string;
    readonly comment: string;
    readonly recordCount: number;
    readonly lastRecordTime: number;
    readonly balances: SocialGiftBalanceResponse[];
}

export interface SocialGiftContactPageResponse {
    readonly items: SocialGiftContactInfoResponse[];
    readonly totalCount: number;
}

export interface SocialGiftRecordInfoResponse {
    readonly id: string;
    readonly contactId: string;
    readonly direction: SocialGiftDirection;
    readonly eventType: SocialGiftEventType;
    readonly eventName: string;
    readonly currency: string;
    readonly transaction: TransactionInfoResponse;
}

export interface SocialGiftRecordPageResponse {
    readonly items: SocialGiftRecordInfoResponse[];
    readonly totalCount: number;
}

export interface SocialGiftSummaryResponse {
    readonly balances: SocialGiftBalanceResponse[];
    readonly receivableContactCount: number;
    readonly payableContactCount: number;
}
