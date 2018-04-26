package clients

import (
	"context"
	"net/url"

	"git.containerum.net/ch/cherry"
	"git.containerum.net/ch/cherry/adaptors/cherrylog"
	"git.containerum.net/ch/kube-api/pkg/model"
	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type KubeAPIClient interface {
	CreateNamespace(ctx context.Context, req model.NamespaceWithOwner) error
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
