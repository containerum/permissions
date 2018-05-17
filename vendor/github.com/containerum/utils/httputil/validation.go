package httputil

import (
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/gonic"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
)

// ValidateQueryParamsMiddleware validates query parameters with provided tags. Key of "vmap" is parameter name, value is tag
func ValidateQueryParamsMiddleware(vmap map[string]string, validate *validator.Validate, translator *ut.UniversalTranslator, validationErr cherry.ErrConstruct) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		for param, tag := range vmap {
			for _, value := range ctx.Request.URL.Query()[param] {
				vErr := validate.VarCtx(ctx.Request.Context(), value, tag)
				t, _ := translator.FindTranslator(GetAcceptedLanguages(ctx.Request.Context())...)
				err := validationErr().AddDetailF("Query parameter \"%s\": %s", param, vErr.(validator.ValidationErrors).Translate(t))
				gonic.Gonic(err, ctx)
				return
			}
		}
	}
}

// ValidateURLParamsMiddleware validates URL parameters with provided tags. Key of "vmap" is parameter name, value is tag
func ValidateURLParamsMiddleware(vmap map[string]string, validate *validator.Validate, translator *ut.UniversalTranslator, validationErr cherry.ErrConstruct) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		for param, tag := range vmap {
			for _, value := range ctx.Param(param) {
				vErr := validate.VarCtx(ctx.Request.Context(), value, tag)
				t, _ := translator.FindTranslator(GetAcceptedLanguages(ctx.Request.Context())...)
				err := validationErr().AddDetailF("URL parameter \"%s\": %s", param, vErr.(validator.ValidationErrors).Translate(t))
				gonic.Gonic(err, ctx)
				return
			}
		}
	}
}
