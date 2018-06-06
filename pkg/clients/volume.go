package clients

import (
	"context"
	"net/url"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/volume-manager/pkg/models"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type VolumeManagerClient interface {
	CreateVolume(ctx context.Context, nsID, label string, capacity int) error
	DeleteNamespaceVolumes(ctx context.Context, nsID string) error
	DeleteAllUserVolumes(ctx context.Context) error
}

type VolumeManagerHTTPClient struct {
	log    *cherrylog.LogrusAdapter
	client *resty.Client
}

func NewVolumeManagerHTTPClient(url *url.URL) *VolumeManagerHTTPClient {
	log := logrus.WithField("component", "volume_manager_client")
	client := resty.New().
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetHostURL(url.String()).
		SetDebug(true).
		SetError(cherry.Err{}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &VolumeManagerHTTPClient{
		log:    cherrylog.NewLogrusAdapter(log),
		client: client,
	}
}

func (v *VolumeManagerHTTPClient) CreateVolume(ctx context.Context, nsID, label string, capacity int) error {
	v.log.WithFields(logrus.Fields{
		"namespace_id": nsID,
		"label":        label,
		"capacity":     capacity,
	}).Debugf("create volume")

	resp, err := v.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(model.DirectVolumeCreateRequest{
			Label:    label,
			Capacity: capacity,
		}).
		SetPathParams(map[string]string{
			"namespace": nsID,
		}).
		Post("/limits/namespaces/{namespace}/volumes")
	if err != nil {
		return errors.ErrInternal().Log(err, v.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (v *VolumeManagerHTTPClient) DeleteNamespaceVolumes(ctx context.Context, nsID string) error {
	v.log.WithField("namespace_id", nsID)

	resp, err := v.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetPathParams(map[string]string{
			"namespace": nsID,
		}).
		Delete("/namespaces/{ns_id}/volumes")
	if err != nil {
		return errors.ErrInternal().Log(err, v.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (v *VolumeManagerHTTPClient) DeleteAllUserVolumes(ctx context.Context) error {
	v.log.Debugf("delete all user volumes")

	resp, err := v.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Delete("/volumes")
	if err != nil {
		return errors.ErrInternal().Log(err, v.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

type VolumeManagerDummyClient struct {
	log *logrus.Entry
}

func NewVolumeManagerDummyClient() *VolumeManagerDummyClient {
	return &VolumeManagerDummyClient{
		log: logrus.WithField("component", "volume_manager_stub"),
	}
}

func (v *VolumeManagerDummyClient) CreateVolume(ctx context.Context, nsID, label string, capacity int) error {
	v.log.WithFields(logrus.Fields{
		"namespace_id": nsID,
		"label":        label,
		"capacity":     capacity,
	}).Debugf("create volume")

	return nil
}

func (v *VolumeManagerDummyClient) DeleteNamespaceVolumes(ctx context.Context, nsID string) error {
	v.log.WithField("namespace_id", nsID)

	return nil
}

func (v *VolumeManagerDummyClient) DeleteAllUserVolumes(ctx context.Context) error {
	v.log.Debugf("delete all user volumes")

	return nil
}
