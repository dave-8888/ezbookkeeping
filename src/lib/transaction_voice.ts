import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';

import { recognizeSpeechOnce } from '@/lib/voice.ts';
import {
    type TransactionTextRecognitionContext,
    recognizeTransactionByText
} from '@/lib/transaction_text.ts';

export interface VoiceTransactionRecognitionContext extends TransactionTextRecognitionContext {
    readonly languageTag: string;
}

export async function recognizeTransactionByVoice(context: VoiceTransactionRecognitionContext): Promise<RecognizedTransactionResponse> {
    const text = await recognizeSpeechOnce(context.languageTag);
    return recognizeTransactionByText(text, context);
}
