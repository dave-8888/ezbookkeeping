package api

import (
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/errs"
	"github.com/mayswind/financially_free/pkg/llm/catalog"
	"github.com/mayswind/financially_free/pkg/models"
	"github.com/mayswind/financially_free/pkg/settings"
)

// UserLLMProvidersHandler returns the current administrator-maintained provider catalog.
func (a *UserLLMSettingsApi) UserLLMProvidersHandler(c *core.WebContext) (any, *errs.Error) {
	providers, err := catalog.Load(settings.Container.GetCurrentConfig().LLMProviderCatalogFile)
	if err != nil {
		return nil, errs.ErrOperationFailed
	}
	return providers, nil
}

// UserLLMModelsHandler uses the form values, retaining saved keys only for the same endpoint.
func (a *UserLLMSettingsApi) UserLLMModelsHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.UserLLMSettingUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)
	if err != nil {
		return nil, errs.ErrUserNotFound
	}
	if user.FeatureRestriction.Contains(core.USER_FEATURE_RESTRICTION_TYPE_CREATE_TRANSACTION_FROM_AI_IMAGE_RECOGNITION) {
		return nil, errs.ErrNotPermittedToPerformThisAction
	}
	saved, err := a.userLLMSettings.GetUserLLMSettingByUid(c, uid)
	if err != nil {
		return nil, errs.ErrOperationFailed
	}
	setting := a.createUserLLMSettingModel(uid, &req, saved)
	if !setting.Enabled {
		return nil, errs.ErrLargeLanguageModelProviderNotEnabled
	}
	config := setting.ToLLMConfig(settings.Container.GetCurrentConfig().ReceiptImageRecognitionLLMConfig)
	result, err := catalog.ListModels(c.Request.Context(), config)
	if err != nil {
		return nil, errs.ErrFailedToRequestRemoteApi
	}
	return result, nil
}
