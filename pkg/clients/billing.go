package clients

import (
	"context"
	"fmt"
	"net/url"

	"git.containerum.net/ch/permissions/pkg/errors"
	berrors "github.com/containerum/bill-external/errors"
	btypes "github.com/containerum/bill-external/models"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

// BillingClient is an interface to billing service
type BillingClient interface {
	Subscribe(ctx context.Context, req btypes.SubscribeTariffRequest) error
	Rename(ctx context.Context, resourceID, newLabel string) error
	UpdateSubscription(ctx context.Context, resourceID, newTariffID string) error
	Unsubscribe(ctx context.Context, resourceID string) error
	MassiveUnsubscribe(ctx context.Context, resourceIDs []string) error

	GetNamespaceTariff(ctx context.Context, tariffID string) (btypes.NamespaceTariff, error)
	GetVolumeTariff(ctx context.Context, tariffID string) (btypes.VolumeTariff, error)
}

// Data for dummy client

type BillingDummyClient struct {
	log *cherrylog.LogrusAdapter
}

var fakeNSData = `
[
  {
    "id": "f3091cc9-6dc3-470e-ac54-84defe011111",
    "created_at": "2017-12-26T13:53:56Z",
    "cpu_limit": 500,
    "memory_limit": 512,
    "traffic": 20,
    "traffic_price": 0.333,
    "external_services": 2,
    "internal_services": 5,
    "is_active": true,
    "is_public": true,
    "price": 0
  },
  {
    "id": "4563e8c1-fb41-416a-9798-e949a2616260",
    "created_at": "2017-12-26T13:57:45Z",
    "cpu_limit": 900,
    "memory_limit": 1024,
    "traffic": 50,
    "traffic_price": 0.5,
    "external_services": 10,
    "internal_services": 20,
    "is_active": true,
    "is_public": true,
    "price": 0
  }
]
`

var fakeVolumeData = `
[
  {
    "id": "15348470-e98f-4da0-8d2e-8c65e15d6eeb",
    "created_at": "2017-12-27T07:55:22Z",
    "storage_limit": 1,
    "replicas_limit": 2,
    "is_persistent": false,
    "is_active": true,
    "is_public": true,
    "price": 0
  },
  {
    "id": "11a35f90-c343-4fc1-a966-381f75568036",
    "created_at": "2017-12-27T07:55:22Z",
    "storage_limit": 2,
    "replicas_limit": 2,
    "is_persistent": false,
    "is_active": true,
    "is_public": true,
    "price": 0
  }
]
`

var (
	fakeNSTariffs     []btypes.NamespaceTariff
	fakeVolumeTariffs []btypes.VolumeTariff
)

func init() {
	var err error
	err = jsoniter.Unmarshal([]byte(fakeNSData), &fakeNSTariffs)
	if err != nil {
		panic(err)
	}
	err = jsoniter.Unmarshal([]byte(fakeVolumeData), &fakeVolumeTariffs)
	if err != nil {
		panic(err)
	}
}

type BillingHTTPClient struct {
	client *resty.Client
	log    *cherrylog.LogrusAdapter
}

func NewBillingHTTPClient(u *url.URL) *BillingHTTPClient {
	log := logrus.WithField("component", "billing_client")
	client := resty.New().
		SetHostURL(u.String()).
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetDebug(true).
		SetError(cherry.Err{}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &BillingHTTPClient{
		client: client,
		log:    cherrylog.NewLogrusAdapter(log),
	}
}

func (b *BillingHTTPClient) Subscribe(ctx context.Context, req btypes.SubscribeTariffRequest) error {
	b.log.WithFields(logrus.Fields{
		"tariff_id":   req.TariffID,
		"resource_id": req.ResourceID,
		"kind":        req.ResourceType,
	}).Debugln("subscribing")

	resp, err := b.client.R().
		SetBody(req).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Post("/isp/subscription")
	if err != nil {
		return errors.ErrInternal().Log(err, b.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (b *BillingHTTPClient) Rename(ctx context.Context, resourceID, newLabel string) error {
	b.log.WithFields(logrus.Fields{
		"resource_id": resourceID,
		"new_label":   newLabel,
	}).Debugln("Rename")

	resp, err := b.client.R().
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(btypes.RenameRequest{
			ResourceLabel: newLabel,
		}).
		SetPathParams(map[string]string{
			"resource": resourceID,
		}).
		Put("/resource/{resource}")
	if err != nil {
		return errors.ErrInternal().Log(err, b.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (b *BillingHTTPClient) UpdateSubscription(ctx context.Context, resourceID, newTariffID string) error {
	b.log.WithFields(logrus.Fields{
		"resource_id":   resourceID,
		"new_tariff_id": newTariffID,
	}).Debugf("update subscription")

	resp, err := b.client.R().
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(map[string]string{"tariff_id": newTariffID}).
		SetPathParams(map[string]string{"resource": resourceID}).
		Put("/isp/subscription/{resource}")
	if err != nil {
		return errors.ErrInternal().Log(err, b.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (b *BillingHTTPClient) Unsubscribe(ctx context.Context, resourceID string) error {
	b.log.WithFields(logrus.Fields{
		"resource_id": resourceID,
	}).Debugln("unsubscribing")

	resp, err := b.client.R().
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetPathParams(map[string]string{"resource": resourceID}).
		Delete("/isp/subscription/{resource}")
	if err != nil {
		return errors.ErrInternal().Log(err, b.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (b *BillingHTTPClient) MassiveUnsubscribe(ctx context.Context, resourceIDs []string) error {
	b.log.WithField("resource_ids", resourceIDs).Debugln("massive unsubscribing")

	resp, err := b.client.R().
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetBody(btypes.MassiveUnsubscribeTariffRequest{
			Resources: resourceIDs,
		}).
		Delete("/isp/subscription")
	if err != nil {
		return errors.ErrInternal().Log(err, b.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}

	return nil
}

func (b *BillingHTTPClient) GetNamespaceTariff(ctx context.Context, tariffID string) (btypes.NamespaceTariff, error) {
	b.log.WithField("tariff_id", tariffID).Debugln("get namespace tariff")

	resp, err := b.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetResult(btypes.NamespaceTariff{}).
		SetPathParams(map[string]string{
			"tariff": tariffID,
		}).
		Get("/tariffs/namespace/{tariff}")
	if err != nil {
		return btypes.NamespaceTariff{}, errors.ErrInternal().Log(err, b.log)
	}
	if resp.Error() != nil {
		return btypes.NamespaceTariff{}, resp.Error().(*cherry.Err)
	}

	return *resp.Result().(*btypes.NamespaceTariff), nil
}

func (b *BillingHTTPClient) GetVolumeTariff(ctx context.Context, tariffID string) (btypes.VolumeTariff, error) {
	b.log.WithField("tariff_id", tariffID).Debugln("get volume tariff")

	resp, err := b.client.R().
		SetContext(ctx).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetResult(btypes.VolumeTariff{}).
		SetPathParams(map[string]string{
			"tariff": tariffID,
		}).
		Get("/tariffs/volume/{tariff}")
	if err != nil {
		return btypes.VolumeTariff{}, errors.ErrInternal().Log(err, b.log)
	}
	if resp.Error() != nil {
		return btypes.VolumeTariff{}, resp.Error().(*cherry.Err)
	}

	return *resp.Result().(*btypes.VolumeTariff), nil
}

func (b BillingHTTPClient) String() string {
	return fmt.Sprintf("billing service http client: url=%s", b.client.HostURL)
}

// NewDummyBilling creates a dummy billing service client. It does nothing but logs actions.
func NewBillingDummyClient() BillingDummyClient {
	return BillingDummyClient{
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "billing_dummy")),
	}
}

func (b BillingDummyClient) Subscribe(ctx context.Context, req btypes.SubscribeTariffRequest) error {
	b.log.WithFields(logrus.Fields{
		"tariff_id":   req.TariffID,
		"resource_id": req.ResourceID,
		"kind":        req.ResourceType,
	}).Debugln("subscribing")
	return nil
}

func (b BillingDummyClient) Rename(ctx context.Context, resourceID, newLabel string) error {
	b.log.WithFields(logrus.Fields{
		"resource_id": resourceID,
		"new_label":   newLabel,
	}).Debugln("Rename")

	return nil
}

func (b BillingDummyClient) UpdateSubscription(ctx context.Context, resourceID, newTariffID string) error {
	b.log.WithFields(logrus.Fields{
		"resource_id":   resourceID,
		"new_tariff_id": newTariffID,
	}).Debugf("update subscription")

	return nil
}

func (b BillingDummyClient) Unsubscribe(ctx context.Context, resourceID string) error {
	b.log.WithFields(logrus.Fields{
		"resource_id": resourceID,
	}).Debugln("unsubscribing")
	return nil
}

func (b BillingDummyClient) MassiveUnsubscribe(ctx context.Context, resourceIDs []string) error {
	b.log.WithField("resource_ids", resourceIDs).Debugln("massive unsubscribing")

	return nil
}

func (b BillingDummyClient) GetNamespaceTariff(ctx context.Context, tariffID string) (btypes.NamespaceTariff, error) {
	b.log.WithField("tariff_id", tariffID).Debugln("get namespace tariff")
	for _, nsTariff := range fakeNSTariffs {
		if nsTariff.ID != "" && nsTariff.ID == tariffID {
			return nsTariff, nil
		}
	}
	return btypes.NamespaceTariff{}, berrors.ErrNotFound().AddDetailF("namespace tariff %s not exists", tariffID)
}

func (b BillingDummyClient) GetVolumeTariff(ctx context.Context, tariffID string) (btypes.VolumeTariff, error) {
	b.log.WithField("tariff_id", tariffID).Debugln("get volume tariff")
	for _, volumeTariff := range fakeVolumeTariffs {
		if volumeTariff.ID != "" && volumeTariff.ID == tariffID {
			return volumeTariff, nil
		}
	}
	return btypes.VolumeTariff{}, berrors.ErrNotFound().AddDetailF("volume tariff %s not exists", tariffID)
}

func (b BillingDummyClient) String() string {
	return "billing service dummy client"
}
