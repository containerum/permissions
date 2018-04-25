package router

import (
	"net/textproto"

	"git.containerum.net/ch/api-gateway/pkg/utils/headers"
	"git.containerum.net/ch/kube-client/pkg/cherry"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/static"
	"git.containerum.net/ch/utils/httputil"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
)

type TranslateValidate struct {
	*ut.UniversalTranslator
	*validator.Validate
}

func (tv *TranslateValidate) HandleError(err error) (int, *cherry.Err) {
	switch err.(type) {
	case *cherry.Err:
		e := err.(*cherry.Err)
		return e.StatusHTTP, e
	default:
		return errors.ErrInternal().StatusHTTP, errors.ErrInternal().AddDetailsErr(err)
	}
}

func (tv *TranslateValidate) BadRequest(ctx *gin.Context, err error) (int, *cherry.Err) {
	if validationErr, ok := err.(validator.ValidationErrors); ok {
		ret := errors.ErrRequestValidationFailed()
		for _, fieldErr := range validationErr {
			if fieldErr == nil {
				continue
			}
			t, _ := tv.FindTranslator(httputil.GetAcceptedLanguages(ctx.Request.Context())...)
			ret.AddDetailF("Field %s: %s", fieldErr.Namespace(), fieldErr.Translate(t))
		}
		return ret.StatusHTTP, ret
	}
	return errors.ErrRequestValidationFailed().StatusHTTP, errors.ErrRequestValidationFailed().AddDetailsErr(err)
}

func (tv *TranslateValidate) ValidateHeaders(headerTagMap map[string]string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		headerErr := make(map[string]validator.ValidationErrors)
		for header, tag := range headerTagMap {
			ferr := tv.VarCtx(ctx.Request.Context(), ctx.GetHeader(textproto.CanonicalMIMEHeaderKey(header)), tag)
			if ferr != nil {
				headerErr[header] = ferr.(validator.ValidationErrors)
			}
		}
		if len(headerErr) > 0 {
			ret := errors.ErrRequestValidationFailed()
			for header, fieldErrs := range headerErr {
				for _, fieldErr := range fieldErrs {
					if fieldErr == nil {
						continue
					}
					t, _ := tv.FindTranslator(httputil.GetAcceptedLanguages(ctx.Request.Context())...)
					ret.AddDetailF("Header %s: %s", header, fieldErr.Translate(t))
				}
			}
			ctx.AbortWithStatusJSON(ret.StatusHTTP, ret)
			return
		}
	}
}

type Router struct {
	engine gin.IRouter
	tv     *TranslateValidate
}

func NewRouter(engine gin.IRouter, tv *TranslateValidate) *Router {
	corsCfg := cors.DefaultConfig()
	corsCfg.AllowAllOrigins = true
	corsCfg.AddAllowHeaders(
		headers.UserIDXHeader,
		headers.UserAgentXHeader,
	)
	engine.Use(cors.New(corsCfg))
	engine.StaticFS("/static", static.HTTP)

	ret := &Router{
		engine: engine,
		tv:     tv,
	}
	ret.engine.Use(httputil.SaveHeaders)
	ret.engine.Use(httputil.PrepareContext)
	ret.engine.Use(httputil.RequireHeaders(errors.ErrRequiredHeadersNotProvided, headers.UserIDXHeader, headers.UserRoleXHeader))
	ret.engine.Use(tv.ValidateHeaders(map[string]string{
		headers.UserIDXHeader:   "uuid",
		headers.UserRoleXHeader: "eq=admin|eq=user",
	}))
	ret.engine.Use(httputil.SubstituteUserMiddleware(tv.Validate, tv.UniversalTranslator, errors.ErrRequestValidationFailed))
	return ret
}
