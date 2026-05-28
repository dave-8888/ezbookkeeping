import { CategoryType } from '@/core/category.ts';
import { TransactionType } from '@/core/transaction.ts';

import type { Account } from '@/models/account.ts';
import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';
import type { TransactionTag } from '@/models/transaction_tag.ts';

interface MutableRecognizedTransactionResponse {
    type?: number;
    time?: number;
    categoryId?: string;
    sourceAccountId?: string;
    destinationAccountId?: string;
    sourceAmount?: number;
    destinationAmount?: number;
    tagIds?: string[];
    comment?: string;
}

export interface TransactionTextParseContext {
    readonly allCategories: Record<number, TransactionCategory[]>;
    readonly allVisibleAccounts: Account[];
    readonly allTags: TransactionTag[];
    readonly currentUnixTime?: number;
}

export interface TransactionTextRecognitionContext extends TransactionTextParseContext {
    readonly recognizeTransactionText?: (text: string) => Promise<RecognizedTransactionResponse>;
}

interface MatchCandidate<T> {
    readonly item: T;
    readonly name: string;
    readonly index: number;
}

interface CategoryMatch {
    readonly id: string;
    readonly type: CategoryType;
}

export async function recognizeTransactionByText(text: string, context: TransactionTextRecognitionContext): Promise<RecognizedTransactionResponse> {
    if (context.recognizeTransactionText) {
        try {
            const recognized = await context.recognizeTransactionText(text);
            return withRecognizedTextComment(recognized, text);
        } catch {
            // Fall back to deterministic local parsing when the LLM route is disabled or fails.
        }
    }

    return parseTransactionTextLocally(text, context);
}

export function parseTransactionTextLocally(text: string, context: TransactionTextParseContext): RecognizedTransactionResponse {
    const response: MutableRecognizedTransactionResponse = {
        comment: text.trim()
    };

    const typeFromText = parseTransactionType(text);
    const categoryMatch = findBestCategoryMatch(text, context.allCategories, typeFromText);

    if (categoryMatch) {
        response.categoryId = categoryMatch.id;
        response.type = categoryTypeToTransactionType(categoryMatch.type);
    } else if (typeFromText) {
        response.type = typeFromText;
    }

    const amount = parseAmount(text);

    if (typeof amount === 'number') {
        response.sourceAmount = amount;
    }

    const transactionTime = parseTransactionTime(text, context.currentUnixTime);

    if (typeof transactionTime === 'number') {
        response.time = transactionTime;
    }

    const accountMatches = findAccountMatches(text, context.allVisibleAccounts);

    if (accountMatches.length > 0 && accountMatches[0]) {
        response.sourceAccountId = accountMatches[0].item.id;
    }

    if ((response.type === TransactionType.Transfer || typeFromText === TransactionType.Transfer) && accountMatches.length > 1 && accountMatches[1]) {
        response.destinationAccountId = accountMatches[1].item.id;
    }

    const tagIds = findTagIds(text, context.allTags);

    if (tagIds.length > 0) {
        response.tagIds = tagIds;
    }

    return response;
}

export function getRecognizedTransactionQueryParams(result: RecognizedTransactionResponse): string[] {
    const params: string[] = [];

    if (result.type) {
        params.push(`type=${result.type}`);
    }

    if (result.time) {
        params.push(`time=${result.time}`);
    }

    if (result.categoryId) {
        params.push(`categoryId=${result.categoryId}`);
    }

    if (result.sourceAccountId) {
        params.push(`accountId=${result.sourceAccountId}`);
    }

    if (result.destinationAccountId) {
        params.push(`destinationAccountId=${result.destinationAccountId}`);
    }

    if (typeof result.sourceAmount === 'number') {
        params.push(`amount=${result.sourceAmount}`);
    }

    if (typeof result.destinationAmount === 'number') {
        params.push(`destinationAmount=${result.destinationAmount}`);
    }

    if (result.tagIds) {
        params.push(`tagIds=${result.tagIds.join(',')}`);
    }

    if (result.comment) {
        params.push(`comment=${encodeURIComponent(result.comment)}`);
    }

    return params;
}

export function hasRecognizedTransactionFields(result: RecognizedTransactionResponse): boolean {
    return !!result.type ||
        !!result.time ||
        !!result.categoryId ||
        !!result.sourceAccountId ||
        !!result.destinationAccountId ||
        typeof result.sourceAmount === 'number' ||
        typeof result.destinationAmount === 'number' ||
        !!(result.tagIds && result.tagIds.length);
}

function withRecognizedTextComment(result: RecognizedTransactionResponse, text: string): RecognizedTransactionResponse {
    if (result.comment) {
        return result;
    }

    return {
        ...result,
        comment: text.trim()
    };
}

function parseTransactionType(text: string): TransactionType | undefined {
    const lowerText = text.toLowerCase();

    if (containsAny(lowerText, ['transfer', 'move money', '转账', '转入', '转出', '转给', '转到'])) {
        return TransactionType.Transfer;
    }

    if (containsAny(lowerText, ['income', 'salary', 'refund', 'reimbursement', 'deposit', 'received', '收入', '工资', '奖金', '报销', '退款', '收款', '到账'])) {
        return TransactionType.Income;
    }

    if (containsAny(lowerText, ['expense', 'spent', 'paid', 'payment', 'buy', 'bought', '支出', '消费', '付款', '支付', '买', '花', '午饭', '晚饭', '早餐', '吃饭', '打车'])) {
        return TransactionType.Expense;
    }

    return undefined;
}

function containsAny(text: string, keywords: string[]): boolean {
    for (const keyword of keywords) {
        if (text.indexOf(keyword) >= 0) {
            return true;
        }
    }

    return false;
}

function parseAmount(text: string): number | undefined {
    const amountRegex = /([¥￥$])?\s*([+-]?\d+(?:[\.,]\d{1,2})?)\s*(元|块|块钱|yuan|rmb|cny|dollars?|usd)?/gi;
    const candidates: { value: number; score: number; index: number }[] = [];

    let match: RegExpExecArray | null;

    while ((match = amountRegex.exec(text)) !== null) {
        const amountText = match[2];

        if (!amountText || isDateOrTimeAdjacent(text, match.index, amountRegex.lastIndex)) {
            continue;
        }

        const numericValue = Number.parseFloat(amountText.replace(',', '.'));

        if (!Number.isFinite(numericValue)) {
            continue;
        }

        const hasCurrencyMarker = !!match[1] || !!match[3];

        candidates.push({
            value: Math.round(numericValue * 100),
            score: hasCurrencyMarker ? 2 : 1,
            index: match.index
        });
    }

    if (!candidates.length) {
        return undefined;
    }

    candidates.sort((item1, item2) => item2.score - item1.score || item2.index - item1.index);
    return candidates[0]?.value;
}

function isDateOrTimeAdjacent(text: string, start: number, end: number): boolean {
    const previousChar = start > 0 ? text[start - 1] : '';
    const nextChar = end < text.length ? text[end] : '';

    return !!previousChar && /[-/:：年月日号点时分秒]/.test(previousChar) || !!nextChar && /[-/:：年月日号点时分秒]/.test(nextChar);
}

function parseTransactionTime(text: string, currentUnixTime?: number): number | undefined {
    const baseDate = currentUnixTime ? new Date(currentUnixTime * 1000) : new Date();
    let dateMatched = false;
    let timeMatched = false;
    let year = baseDate.getFullYear();
    let month = baseDate.getMonth();
    let day = baseDate.getDate();
    let hour = 0;
    let minute = 0;
    let second = 0;

    const explicitDateMatch = text.match(/(\d{4})[-\/年](\d{1,2})[-\/月](\d{1,2})[日号]?/);

    if (explicitDateMatch && explicitDateMatch[1] && explicitDateMatch[2] && explicitDateMatch[3]) {
        year = Number.parseInt(explicitDateMatch[1], 10);
        month = Number.parseInt(explicitDateMatch[2], 10) - 1;
        day = Number.parseInt(explicitDateMatch[3], 10);
        dateMatched = true;
    } else {
        const monthDayMatch = text.match(/(\d{1,2})月(\d{1,2})[日号]?/);

        if (monthDayMatch && monthDayMatch[1] && monthDayMatch[2]) {
            month = Number.parseInt(monthDayMatch[1], 10) - 1;
            day = Number.parseInt(monthDayMatch[2], 10);
            dateMatched = true;
        } else if (containsAny(text, ['前天', 'day before yesterday'])) {
            baseDate.setDate(baseDate.getDate() - 2);
            year = baseDate.getFullYear();
            month = baseDate.getMonth();
            day = baseDate.getDate();
            dateMatched = true;
        } else if (containsAny(text, ['昨天', 'yesterday'])) {
            baseDate.setDate(baseDate.getDate() - 1);
            year = baseDate.getFullYear();
            month = baseDate.getMonth();
            day = baseDate.getDate();
            dateMatched = true;
        } else if (containsAny(text, ['今天', 'today'])) {
            dateMatched = true;
        }
    }

    const colonTimeMatch = text.match(/(\d{1,2})[:：](\d{1,2})/);

    if (colonTimeMatch && colonTimeMatch[1] && colonTimeMatch[2]) {
        hour = Number.parseInt(colonTimeMatch[1], 10);
        minute = Number.parseInt(colonTimeMatch[2], 10);
        timeMatched = true;
    } else {
        const textualTimeMatch = text.match(/(凌晨|早上|上午|中午|下午|晚上)?\s*(\d{1,2})\s*[点时](?:\s*(\d{1,2})\s*分?)?/);

        if (textualTimeMatch && textualTimeMatch[2]) {
            const dayPart = textualTimeMatch[1] || '';
            hour = Number.parseInt(textualTimeMatch[2], 10);
            minute = textualTimeMatch[3] ? Number.parseInt(textualTimeMatch[3], 10) : 0;

            if ((dayPart === '下午' || dayPart === '晚上') && hour < 12) {
                hour += 12;
            } else if (dayPart === '凌晨' && hour === 12) {
                hour = 0;
            } else if (dayPart === '中午' && hour < 11) {
                hour += 12;
            }

            timeMatched = true;
        }
    }

    if (!dateMatched && !timeMatched) {
        return undefined;
    }

    const parsedDate = new Date(year, month, day, hour, minute, second, 0);
    const parsedUnixTime = Math.floor(parsedDate.getTime() / 1000);

    return Number.isFinite(parsedUnixTime) ? parsedUnixTime : undefined;
}

function findBestCategoryMatch(text: string, allCategories: Record<number, TransactionCategory[]>, preferredType?: TransactionType): CategoryMatch | undefined {
    const preferredCategoryType = preferredType ? transactionTypeToCategoryType(preferredType) : undefined;
    const candidates: MatchCandidate<CategoryMatch>[] = [];

    for (const categoryType of [CategoryType.Expense, CategoryType.Income, CategoryType.Transfer]) {
        const categories = allCategories[categoryType] || [];

        for (const category of categories) {
            if (category.hidden) {
                continue;
            }

            if (category.name && text.indexOf(category.name) >= 0) {
                const firstSubCategory = getFirstVisibleSubCategory(category);

                if (firstSubCategory) {
                    candidates.push({
                        item: {
                            id: firstSubCategory.id,
                            type: categoryType
                        },
                        name: category.name,
                        index: text.indexOf(category.name)
                    });
                }
            }

            for (const subCategory of category.subCategories || []) {
                if (subCategory.hidden || !subCategory.name || text.indexOf(subCategory.name) < 0) {
                    continue;
                }

                candidates.push({
                    item: {
                        id: subCategory.id,
                        type: categoryType
                    },
                    name: subCategory.name,
                    index: text.indexOf(subCategory.name)
                });
            }
        }
    }

    if (!candidates.length) {
        return undefined;
    }

    candidates.sort((item1, item2) => {
        const preferredScore1 = preferredCategoryType && item1.item.type === preferredCategoryType ? 1 : 0;
        const preferredScore2 = preferredCategoryType && item2.item.type === preferredCategoryType ? 1 : 0;
        return preferredScore2 - preferredScore1 || item2.name.length - item1.name.length || item1.index - item2.index;
    });

    return candidates[0]?.item;
}

function getFirstVisibleSubCategory(category: TransactionCategory): TransactionCategory | undefined {
    for (const subCategory of category.subCategories || []) {
        if (!subCategory.hidden) {
            return subCategory;
        }
    }

    return undefined;
}

function findAccountMatches(text: string, allVisibleAccounts: Account[]): MatchCandidate<Account>[] {
    const candidates: MatchCandidate<Account>[] = [];

    for (const account of allVisibleAccounts) {
        if (account.hidden || !account.name) {
            continue;
        }

        const index = text.indexOf(account.name);

        if (index < 0) {
            continue;
        }

        candidates.push({
            item: account,
            name: account.name,
            index: index
        });
    }

    candidates.sort((item1, item2) => item1.index - item2.index || item2.name.length - item1.name.length);

    const matches: MatchCandidate<Account>[] = [];

    for (const candidate of candidates) {
        if (hasOverlappedMatch(candidate, matches)) {
            continue;
        }

        matches.push(candidate);
    }

    return matches;
}

function hasOverlappedMatch<T>(candidate: MatchCandidate<T>, matches: MatchCandidate<T>[]): boolean {
    const candidateStart = candidate.index;
    const candidateEnd = candidate.index + candidate.name.length;

    for (const match of matches) {
        const matchStart = match.index;
        const matchEnd = match.index + match.name.length;

        if (candidateStart < matchEnd && candidateEnd > matchStart) {
            return true;
        }
    }

    return false;
}

function findTagIds(text: string, allTags: TransactionTag[]): string[] {
    const tagMatches: MatchCandidate<TransactionTag>[] = [];

    for (const tag of allTags) {
        if (tag.hidden || !tag.name) {
            continue;
        }

        const index = text.indexOf(tag.name);

        if (index < 0) {
            continue;
        }

        tagMatches.push({
            item: tag,
            name: tag.name,
            index: index
        });
    }

    tagMatches.sort((item1, item2) => item1.index - item2.index || item2.name.length - item1.name.length);
    return tagMatches.map(match => match.item.id);
}

function categoryTypeToTransactionType(categoryType: CategoryType): TransactionType {
    if (categoryType === CategoryType.Income) {
        return TransactionType.Income;
    } else if (categoryType === CategoryType.Transfer) {
        return TransactionType.Transfer;
    } else {
        return TransactionType.Expense;
    }
}

function transactionTypeToCategoryType(transactionType: TransactionType): CategoryType | undefined {
    if (transactionType === TransactionType.Income) {
        return CategoryType.Income;
    } else if (transactionType === TransactionType.Expense) {
        return CategoryType.Expense;
    } else if (transactionType === TransactionType.Transfer) {
        return CategoryType.Transfer;
    }

    return undefined;
}
