package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/gonic"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/router"
	"git.containerum.net/ch/permissions/pkg/utils/validation"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
)

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func main() {
	exitOnError(setupLogger())

	logrus.Info("starting permissions service")

	listenAddr, err := getListenAddr()
	exitOnError(err)

	translate := setupTranslator()
	validate := validation.StandardPermissionsValidator(translate)

	db, err := setupDB()
	exitOnError(err)
	defer db.Close()

	g := gin.New()
	g.Use(gonic.Recovery(errors.ErrInternal, cherrylog.NewLogrusAdapter(logrus.WithField("component", "gin_recovery"))))
	g.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))
	binding.Validator = &validation.GinValidatorV9{Validate: validate} // gin has no local validator

	r := router.NewRouter(g, &router.TranslateValidate{UniversalTranslator: translate, Validate: validate})
	r.SetupAccessRoutes(nil)

	// for graceful shutdown
	httpsrv := &http.Server{
		Addr:    listenAddr,
		Handler: g,
	}

	// serve connections
	go exitOnError(httpsrv.ListenAndServe())

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt) // subscribe on interrupt event
	<-quit                            // wait for event
	logrus.Infoln("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	exitOnError(httpsrv.Shutdown(ctx))
}
