package main

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"

	"git.containerum.net/ch/permissions/pkg/clients"
	"git.containerum.net/ch/permissions/pkg/database"
	"git.containerum.net/ch/permissions/pkg/database/postgres"
	"git.containerum.net/ch/permissions/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/universal-translator"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

type operationMode int

const (
	modeDebug operationMode = iota
	modeRelease
)

var opMode operationMode

func setupLogger(ctx *cli.Context) error {
	mode := ctx.String(ModeFlag.Name)
	switch mode {
	case "debug":
		opMode = modeDebug
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
	case "release", "":
		opMode = modeRelease
		gin.SetMode(gin.ReleaseMode)
		logrus.SetFormatter(&logrus.JSONFormatter{})

		level := logrus.Level(ctx.Int(LogLevelFlag.Name))
		if level > logrus.DebugLevel || level < logrus.PanicLevel {
			return errors.New("invalid log level")
		}
		logrus.SetLevel(level)
	default:
		return errors.New("invalid operation mode (must be 'debug' or 'release')")
	}
	return nil
}

func setupDB(ctx *cli.Context) (database.DB, error) {
	return postgres.Connect(fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		ctx.String(DBUserFlag.Name),
		ctx.String(DBPassFlag.Name),
		ctx.String(DBHostFlag.Name),
		ctx.String(DBBaseFlag.Name),
		func() string {
			if ctx.Bool(DBSSLModeFlag.Name) {
				return "enable"
			}
			return "disable"
		}()))
}

func getListenAddr(ctx *cli.Context) (string, error) {
	return ctx.String(ListenAddrFlag.Name), nil
}

func setupTranslator() *ut.UniversalTranslator {
	return ut.New(en.New(), en.New(), en_US.New())
}

func setupAuthClient(addr string) (clients.AuthClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewAuthDummyClient(), nil
	case addr != "":
		return clients.NewAuthGRPCClient(addr)
	default:
		return nil, errors.New("missing configuration for auth service")
	}
}

func setupUserClient(addr string) (clients.UserManagerClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewUserManagerDummyClient(), nil
	case addr != "":
		return clients.NewUserManagerHTTPClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for user-manager service")
	}
}

func setupKubeClient(addr string) (clients.KubeAPIClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewKubeAPIDummyClient(), nil
	case addr != "":
		return clients.NewKubeAPIHTTPClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for kube-api service")
	}
}

func setupResourceClient(addr string) (clients.ResourceServiceClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewResourceServiceDummyClient(), nil
	case addr != "":
		return clients.NewResourceServiceHTTPClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for resource-service")
	}
}

func setupBillingClient(addr string) (clients.BillingClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewBillingDummyClient(), nil
	case addr != "":
		return clients.NewBillingHTTPClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for billing service")
	}
}

func SetupVolumeClient(addr string) (clients.VolumeManagerClient, error) {
	switch {
	case opMode == modeDebug && addr == "":
		return clients.NewVolumeManagerDummyClient(), nil
	case addr != "":
		return clients.NewVolumeManagerHTTPClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for volume-manager service")
	}
}

func SetupSolutionsClient(addr string) (clients.SolutionsClient, error) {
	switch {
	case addr == "":
		return clients.NewSolutionsDummyClient(), nil
	case addr != "":
		return clients.NewSolutionsHTTPClient(&url.URL{Scheme: "http", Host: addr}), nil
	default:
		return nil, errors.New("missing configuration for solutions service")
	}
}

func setupServiceClients(ctx *cli.Context) (*server.Clients, error) {
	var errs []error
	var clients server.Clients
	var err error

	if clients.Auth, err = setupAuthClient(ctx.String(AuthAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}

	if clients.User, err = setupUserClient(ctx.String(UserAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}

	if clients.Kube, err = setupKubeClient(ctx.String(KubeAPIAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}

	if clients.Resource, err = setupResourceClient(ctx.String(ResourceServiceAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}

	if clients.Billing, err = setupBillingClient(ctx.String(BillingAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}

	if clients.Volume, err = SetupVolumeClient(ctx.String(VolumeManagerAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}

	if clients.Solutions, err = SetupSolutionsClient(ctx.String(SolutionsAddrFlag.Name)); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("clients setup errors: %v", errs)
	}

	v := reflect.ValueOf(clients)
	for i := 0; i < reflect.TypeOf(clients).NumField(); i++ {
		f := v.Field(i)
		if str, ok := f.Interface().(fmt.Stringer); ok {
			logrus.Infof("%s", str)
		}
	}

	return &clients, nil
}
