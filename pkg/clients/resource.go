package clients

import (
	"context"
	"fmt"
	"net/url"

	"github.com/containerum/cherry"
	"github.com/containerum/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type ResourceServiceClient interface {
	DeleteNamespaceResources(ctx context.Context, namespaceID string) error
}

type ResourceServiceHTTPClient struct {
	log    *logrus.Entry
	client *resty.Client
}

func NewResourceServiceHTTPClient(url *url.URL) *ResourceServiceHTTPClient {
	log := logrus.WithField("component", "resource_service_client")
	client := resty.New().
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetHostURL(url.String()).
		SetDebug(true).
		SetError(cherry.Err{}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &ResourceServiceHTTPClient{
		log:    log,
		client: client,
	}
}

func (r *ResourceServiceHTTPClient) DeleteNamespaceResources(ctx context.Context, namespaceID string) error {
	r.log.WithField("namespace_id", namespaceID).Debugf("delete namespace resources")

	resp, err := r.client.R().
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Delete(fmt.Sprintf("/namespaces/%s", namespaceID))
	if err != nil {
		return err
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

type ResourceServiceDummyClient struct {
	log *logrus.Entry
}

func NewResourceServiceDummyClient() *ResourceServiceDummyClient {
	return &ResourceServiceDummyClient{
		log: logrus.WithField("component", "resource_service_stub"),
	}
}

func (r *ResourceServiceDummyClient) DeleteNamespaceResources(ctx context.Context, namespaceID string) error {
	r.log.WithField("namespace_id", namespaceID).Debugf("delete namespace resources")

	return nil
}