interface BrowserSpeechRecognitionAlternative {
    readonly transcript: string;
}

interface BrowserSpeechRecognitionResult {
    readonly length: number;
    item(index: number): BrowserSpeechRecognitionAlternative;
}

interface BrowserSpeechRecognitionResultList {
    readonly length: number;
    item(index: number): BrowserSpeechRecognitionResult;
}

interface BrowserSpeechRecognitionEvent extends Event {
    readonly resultIndex: number;
    readonly results: BrowserSpeechRecognitionResultList;
}

interface BrowserSpeechRecognitionErrorEvent extends Event {
    readonly error: string;
    readonly message?: string;
}

interface BrowserSpeechRecognition {
    continuous: boolean;
    interimResults: boolean;
    lang: string;
    maxAlternatives: number;
    onend: (() => void) | null;
    onerror: ((event: BrowserSpeechRecognitionErrorEvent) => void) | null;
    onresult: ((event: BrowserSpeechRecognitionEvent) => void) | null;
    abort(): void;
    start(): void;
    stop(): void;
}

interface BrowserSpeechRecognitionConstructor {
    new(): BrowserSpeechRecognition;
}

type SpeechRecognitionWindow = Window & {
    SpeechRecognition?: BrowserSpeechRecognitionConstructor;
    webkitSpeechRecognition?: BrowserSpeechRecognitionConstructor;
};

function getSpeechRecognitionConstructor(): BrowserSpeechRecognitionConstructor | undefined {
    const speechWindow = window as SpeechRecognitionWindow;
    return speechWindow.SpeechRecognition || speechWindow.webkitSpeechRecognition;
}

export function isSpeechRecognitionSupported(): boolean {
    return !!getSpeechRecognitionConstructor();
}

export function recognizeSpeechOnce(languageTag: string): Promise<string> {
    const SpeechRecognition = getSpeechRecognitionConstructor();

    if (!SpeechRecognition) {
        return Promise.reject({ message: 'Speech recognition is not supported by this browser' });
    }

    return new Promise((resolve, reject) => {
        const recognition = new SpeechRecognition();
        let settled = false;
        let transcript = '';

        const settle = (callback: () => void) => {
            if (settled) {
                return;
            }

            settled = true;
            recognition.onresult = null;
            recognition.onerror = null;
            recognition.onend = null;
            callback();
        };

        recognition.lang = languageTag || window.navigator.language || 'en-US';
        recognition.continuous = false;
        recognition.interimResults = false;
        recognition.maxAlternatives = 1;

        recognition.onresult = (event: BrowserSpeechRecognitionEvent) => {
            for (let i = event.resultIndex; i < event.results.length; i++) {
                const result = event.results.item(i);

                if (result.length < 1) {
                    continue;
                }

                transcript += result.item(0).transcript;
            }
        };

        recognition.onerror = (event: BrowserSpeechRecognitionErrorEvent) => {
            settle(() => reject({ message: event.message || event.error || 'Unable to recognize speech' }));
        };

        recognition.onend = () => {
            const finalTranscript = transcript.trim();

            settle(() => {
                if (finalTranscript) {
                    resolve(finalTranscript);
                } else {
                    reject({ message: 'Unable to recognize speech' });
                }
            });
        };

        try {
            recognition.start();
        } catch (ex) {
            settle(() => reject(ex));
        }
    });
}
