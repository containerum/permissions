package clients

import (
	"context"
	"net/url"

	"git.containerum.net/ch/kube-api/pkg/model"
	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type KubeAPIClient interface {
	CreateNamespace(ctx context.Context, req model.NamespaceWithOwner) error
	SetNamespaceQuota(ctx context.Context, ns model.NamespaceWithOwner) error
	DeleteNamespace(ctx context.Context, ns model.NamespaceWithOwner) error
}

type KubeAPIHTTPClient struct {
	log    *cherrylog.LogrusAdapter
	client *resty.Client
}

func NewKubeAPIHTTPClient(url *url.URL) *KubeAPIHTTPClient {
	log := logrus.WithField("component", "kube_api_client")

	client := resty.New().
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetHostURL(url.String()).
		SetDebug(true).
		SetError(cherry.Err{}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &KubeAPIHTTPClient{
		log:    cherrylog.NewLogrusAdapter(log),
		client: client,
	}
}

func (k *KubeAPIHTTPClient) CreateNamespace(ctx context.Context, req model.NamespaceWithOwner) error {
	k.log.WithFields(logrus.Fields{
		"cpu":    req.Resources.Hard.CPU,
		"memory": req.Resources.Hard.Memory,
		"name":   req.Label,
		"access": req.Access,
	}).Debug("create namespace")

	resp, err := k.client.R().
		SetBody(req).
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Post("/namespaces")
	if err != nil {
		return errors.ErrInternal().Log(err, k.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (k *KubeAPIHTTPClient) SetNamespaceQuota(ctx context.Context, ns model.NamespaceWithOwner) error {
	k.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"label":  ns.Label,
	}).Debug("set namespace quota")

	resp, err := k.client.R().
		SetBody(ns).
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Put("/namespaces/" + url.PathEscape(ns.Label))
	if err != nil {
		return errors.ErrInternal().Log(err, k.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (k *KubeAPIHTTPClient) DeleteNamespace(ctx context.Context, ns model.NamespaceWithOwner) error {
	k.log.WithField("name", ns.Name).Debugf("delete namespace")

	resp, err := k.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Delete("/namespaces/" + ns.Name)
	if err != nil {
		return errors.ErrInternal().Log(err, k.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

type KubeAPIDummyClient struct {
	log *logrus.Entry
}

func NewKubeAPIDummyClient() *KubeAPIDummyClient {
	return &KubeAPIDummyClient{
		log: logrus.WithField("component", "kube_api_client"),
	}
}

func (k *KubeAPIDummyClient) CreateNamespace(ctx context.Context, req model.NamespaceWithOwner) error {
	k.log.WithFields(logrus.Fields{
		"cpu":    req.Resources.Hard.CPU,
		"memory": req.Resources.Hard.Memory,
		"name":   req.Label,
		"access": req.Access,
	}).Debug("create namespace")

	return nil
}

func (k *KubeAPIDummyClient) SetNamespaceQuota(ctx context.Context, ns model.NamespaceWithOwner) error {
	k.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"label":  ns.Label,
	}).Debug("set namespace quota")

	return nil
}

func (k *KubeAPIDummyClient) DeleteNamespace(ctx context.Context, ns model.NamespaceWithOwner) error {
	k.log.WithFields(logrus.Fields{
		"cpu":    ns.Resources.Hard.CPU,
		"memory": ns.Resources.Hard.Memory,
		"label":  ns.Label,
	}).Debug("set namespace quota")

	return nil
}
