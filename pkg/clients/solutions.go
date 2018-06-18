package clients

import (
	"context"
	"fmt"
	"net/url"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/utils/httputil"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type SolutionsClient interface {
	DeleteNamespaceSolutions(ctx context.Context, nsID string) error
}

type SolutionsHTTPClient struct {
	log    *cherrylog.LogrusAdapter
	client *resty.Client
}

func NewSolutionsHTTPClient(url *url.URL) *SolutionsHTTPClient {
	log := cherrylog.NewLogrusAdapter(logrus.WithField("component", "solutions_client"))
	client := resty.New().
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetHostURL(url.String()).
		SetDebug(true).
		SetError(cherry.Err{}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &SolutionsHTTPClient{
		log:    log,
		client: client,
	}
}

func (s *SolutionsHTTPClient) DeleteNamespaceSolutions(ctx context.Context, nsID string) error {
	s.log.WithField("namespace_id", nsID).Debugf("delete namespace solutions")

	resp, err := s.client.R().
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetPathParams(map[string]string{"namespace": nsID}).
		Delete("/namespace/{namespace}/solutions")
	if err != nil {
		return errors.ErrInternal().Log(err, s.log)
	}
	if resp.Error() != nil {
		return resp.Error().(*cherry.Err)
	}
	return nil
}

func (s SolutionsHTTPClient) String() string {
	return fmt.Sprintf("solutions http client: url=%s", s.client.HostURL)
}

type SolutionsDummyClient struct {
	log *cherrylog.LogrusAdapter
}

func NewSolutionsDummyClient() *SolutionsDummyClient {
	return &SolutionsDummyClient{
		log: cherrylog.NewLogrusAdapter(logrus.WithField("component", "solutions_stub")),
	}
}

func (s *SolutionsDummyClient) DeleteNamespaceSolutions(ctx context.Context, nsID string) error {
	s.log.WithField("namespace_id", nsID).Debugf("delete namespace solutions")

	return nil
}

func (s SolutionsDummyClient) String() string {
	return "solutions dummy client"
}
