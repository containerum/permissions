package clients

import (
	"context"
	"fmt"
	"net/url"

	"git.containerum.net/ch/permissions/pkg/errors"
	umtypes "git.containerum.net/ch/user-manager/pkg/models"
	"github.com/containerum/cherry"
	"github.com/containerum/cherry/adaptors/cherrylog"
	"github.com/containerum/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

// UserManagerClient is interface to user-manager service
type UserManagerClient interface {
	UserInfoByLogin(ctx context.Context, login string) (*umtypes.User, error)
	UserInfoByID(ctx context.Context, userID string) (*umtypes.User, error)
	UserLoginIDList(ctx context.Context, userIDs ...string) (map[string]string, error)
}

type UserManagerHTTPClient struct {
	log    *cherrylog.LogrusAdapter
	client *resty.Client
}

// NewUserManagerHTTPClient returns rest-client to user-manager service
func NewUserManagerHTTPClient(url *url.URL) *UserManagerHTTPClient {
	log := logrus.WithField("component", "user_manager_client")
	client := resty.New().
		SetLogger(log.WriterLevel(logrus.DebugLevel)).
		SetHostURL(url.String()).
		SetDebug(true).
		SetError(cherry.Err{}).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")
	client.JSONMarshal = jsoniter.Marshal
	client.JSONUnmarshal = jsoniter.Unmarshal
	return &UserManagerHTTPClient{
		log:    cherrylog.NewLogrusAdapter(log),
		client: client,
	}
}

func (u *UserManagerHTTPClient) UserInfoByLogin(ctx context.Context, login string) (*umtypes.User, error) {
	u.log.WithField("login", login).Debug("get user info by login")
	resp, err := u.client.R().
		SetContext(ctx).
		SetResult(umtypes.User{}).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetPathParams(map[string]string{
			"login": login,
		}).
		Get("/user/info/login/{login}")
	if err != nil {
		return nil, errors.ErrInternal().Log(err, u.log)
	}
	if resp.Error() != nil {
		return nil, resp.Error().(*cherry.Err)
	}
	return resp.Result().(*umtypes.User), nil
}

func (u *UserManagerHTTPClient) UserInfoByID(ctx context.Context, userID string) (*umtypes.User, error) {
	u.log.WithField("id", userID).Debug("get user info by id")
	resp, err := u.client.R().
		SetContext(ctx).
		SetResult(umtypes.User{}).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		SetPathParams(map[string]string{
			"id": userID,
		}).
		Get("/user/info/id/{id}")
	if err != nil {
		return nil, errors.ErrInternal().Log(err, u.log)
	}
	if resp.Error() != nil {
		return nil, resp.Error().(*cherry.Err)
	}
	return resp.Result().(*umtypes.User), nil
}

func (u *UserManagerHTTPClient) UserLoginIDList(ctx context.Context, userIDs ...string) (map[string]string, error) {
	u.log.WithField("user_ids", userIDs).Debug("get users list")
	resp, err := u.client.R().
		SetContext(ctx).
		SetBody(userIDs).
		SetResult(map[string]string{}).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Post("/user/loginid")
	if err != nil {
		return nil, errors.ErrInternal().Log(err, u.log)
	}
	if resp.Error() != nil {
		return nil, resp.Error().(*cherry.Err)
	}

	ret := resp.Result().(*map[string]string)

	return *ret, nil
}

func (u *UserManagerHTTPClient) String() string {
	return fmt.Sprintf("user-manager http client: url=%s", u.client.HostURL)
}

type UserManagerDummyClient struct {
	log         *logrus.Entry
	givenLogins map[string]umtypes.User
}

func NewUserManagerDummyClient() *UserManagerDummyClient {
	return &UserManagerDummyClient{
		log:         logrus.WithField("component", "user_manager_stub"),
		givenLogins: make(map[string]umtypes.User),
	}
}

func (u *UserManagerDummyClient) UserInfoByLogin(ctx context.Context, login string) (*umtypes.User, error) {
	u.log.WithField("id", login).Debug("get user info by login")
	resp, ok := u.givenLogins[login]
	if !ok {
		resp = umtypes.User{
			UserLogin: &umtypes.UserLogin{
				ID:    uuid.NewV4().String(),
				Login: login,
			},
			Role: "user",
			Profile: &umtypes.Profile{
				Data: map[string]interface{}{
					"email": login,
				},
			},
		}
		u.givenLogins[login] = resp
	}
	return &resp, nil
}

func (u *UserManagerDummyClient) UserInfoByID(ctx context.Context, userID string) (*umtypes.User, error) {
	u.log.WithField("id", userID).Debug("get user info by id")
	return &umtypes.User{
		UserLogin: &umtypes.UserLogin{
			ID:    userID,
			Login: "fake-" + userID + "@test.com",
		},
		Role: "user",
		Profile: &umtypes.Profile{
			Data: map[string]interface{}{
				"email": "fake-" + userID + "@test.com",
			},
		},
	}, nil
}

func (u *UserManagerDummyClient) UserLoginIDList(ctx context.Context, userIDs ...string) (map[string]string, error) {
	u.log.Debug("get user info by id")
	ret := make(map[string]string)
	for _, v := range userIDs {
		ret[v] = "fake-" + v + "@test.com"
	}
	return ret, nil
}

func (u *UserManagerDummyClient) String() string {
	return "user-manager dummy client"
}
