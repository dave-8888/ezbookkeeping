export interface RecognizedTransactionResponse {
    readonly type?: number;
    readonly time?: number;
    readonly categoryId?: string;
    readonly sourceAccountId?: string;
    readonly destinationAccountId?: string;
    readonly sourceAmount?: number;
    readonly destinationAmount?: number;
    readonly tagIds?: string[];
    readonly comment?: string;
}

export type RecognizedReceiptImageResponse = RecognizedTransactionResponse;

export interface MCPTransactionResponse {
    readonly status: 'created' | 'needs_input' | 'failed' | 'unknown' | 'processing';
    readonly reply: string;
}
