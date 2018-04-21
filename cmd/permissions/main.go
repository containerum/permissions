package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"text/tabwriter"
	"time"

	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/gonic"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/router"
	"git.containerum.net/ch/permissions/pkg/server"
	"git.containerum.net/ch/permissions/pkg/utils/validation"
	"git.containerum.net/ch/permissions/pkg/utils/version"
	"git.containerum.net/ch/permissions/static"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

//go:generate swagger generate spec -i ../../swagger-basic.yml -o ../../swagger.json

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func errFuture(f func() error) <-chan error {
	ch := make(chan error)
	go func() {
		ch <- f()
	}()
	return ch
}

func prettyPrintFlags(ctx *cli.Context) {
	fmt.Printf("Starting %v %v\n", ctx.App.Name, ctx.App.Version)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent|tabwriter.Debug)
	for _, f := range ctx.App.VisibleFlags() {
		fmt.Fprintf(w, "Flag: %s\t Value: %v\n", f.Names()[0], ctx.Generic(f.Names()[0]))
	}
	w.Flush()
}

const httpServerContextKey = "httpsrv"

func main() {
	app := cli.App{
		Name:        "permissions",
		Description: "Resources permissions management service for Container hosting",
		Version:     version.VERSION,
		Flags: []cli.Flag{
			&ModeFlag,
			&LogLevelFlag,
			&DBAddrFlag,
			&ListenAddrFlag,
		},
		Before: func(ctx *cli.Context) error {
			prettyPrintFlags(ctx)

			if err := setupLogger(ctx); err != nil {
				return err
			}

			listenAddr, err := getListenAddr(ctx)
			if err != nil {
				return err
			}

			translate := setupTranslator()
			validate := validation.StandardPermissionsValidator(translate)

			db, err := setupDB(ctx)
			if err != nil {
				return err
			}

			srv := server.NewServer(db)

			g := gin.New()
			g.Use(gonic.Recovery(errors.ErrInternal, cherrylog.NewLogrusAdapter(logrus.WithField("component", "gin_recovery"))))
			g.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))
			binding.Validator = &validation.GinValidatorV9{Validate: validate} // gin has no local validator

			g.StaticFS("/static", static.HTTP)

			r := router.NewRouter(g, &router.TranslateValidate{UniversalTranslator: translate, Validate: validate})
			r.SetupAccessRoutes(srv)

			// for graceful shutdown
			httpsrv := &http.Server{
				Addr:    listenAddr,
				Handler: g,
			}

			ctx.App.Metadata[httpServerContextKey] = httpsrv

			return nil
		},
		Action: func(ctx *cli.Context) error {
			httpsrv := ctx.App.Metadata[httpServerContextKey].(*http.Server)
			errCh := errFuture(func() error {
				return httpsrv.ListenAndServe()
			})

			// Wait for interrupt signal to gracefully shutdown the server with
			// a timeout of 5 seconds.
			quit := make(chan os.Signal)
			signal.Notify(quit, os.Interrupt, os.Kill) // subscribe on interrupt event

			select {
			case err := <-errCh:
				return err
			case <-quit:
				logrus.Infoln("shutting down server...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return httpsrv.Shutdown(ctx)
			}

			return nil
		},
	}

	exitOnError(app.Run(os.Args))
}
