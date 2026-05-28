export interface UserLLMSettingResponse {
    readonly enabled: boolean;
    readonly llmProvider: string;
    readonly openAIAPIKeySet: boolean;
    readonly openAIModelID: string;
    readonly openAICompatibleBaseURL: string;
    readonly openAICompatibleAPIKeySet: boolean;
    readonly openAICompatibleModelID: string;
    readonly openRouterAPIKeySet: boolean;
    readonly openRouterModelID: string;
    readonly ollamaServerURL: string;
    readonly ollamaModelID: string;
    readonly googleAIAPIKeySet: boolean;
    readonly googleAIModelID: string;
    readonly largeLanguageModelAPIRequestTimeout: number;
    readonly largeLanguageModelAPIProxy: string;
    readonly largeLanguageModelAPISkipTLSVerify: boolean;
    readonly updatedUnixTime: number;
}

export interface UserLLMSettingUpdateRequest {
    enabled: boolean;
    llmProvider: string;
    openAIAPIKey: string;
    openAIModelID: string;
    openAICompatibleBaseURL: string;
    openAICompatibleAPIKey: string;
    openAICompatibleModelID: string;
    openRouterAPIKey: string;
    openRouterModelID: string;
    ollamaServerURL: string;
    ollamaModelID: string;
    googleAIAPIKey: string;
    googleAIModelID: string;
    largeLanguageModelAPIRequestTimeout: number;
    largeLanguageModelAPIProxy: string;
    largeLanguageModelAPISkipTLSVerify: boolean;
}
