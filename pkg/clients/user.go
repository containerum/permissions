package clients

import (
	"context"
	"net/url"

	"git.containerum.net/ch/kube-client/pkg/cherry"
	umtypes "git.containerum.net/ch/user-manager/pkg/models"
	"git.containerum.net/ch/utils/httputil"
	"github.com/json-iterator/go"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

// UserManagerClient is interface to user-manager service
type UserManagerClient interface {
	UserInfoByLogin(ctx context.Context, login string) (*umtypes.User, error)
	UserInfoByID(ctx context.Context, userID string) (*umtypes.User, error)
	UserLoginIDList(ctx context.Context) (map[string]string, error)
}

type httpUserManagerClient struct {
	log    *logrus.Entry
	client *resty.Client
}

// NewHTTPUserManagerClient returns rest-client to user-manager service
func NewHTTPUserManagerClient(url *url.URL) UserManagerClient {
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
	return &httpUserManagerClient{
		log:    log,
		client: client,
	}
}

func (u *httpUserManagerClient) UserInfoByLogin(ctx context.Context, login string) (*umtypes.User, error) {
	u.log.WithField("login", login).Info("get user info by login")
	resp, err := u.client.R().
		SetContext(ctx).
		SetResult(umtypes.User{}).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Get("/user/info/login/" + login)
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, resp.Error().(*cherry.Err)
	}
	return resp.Result().(*umtypes.User), nil
}

func (u *httpUserManagerClient) UserInfoByID(ctx context.Context, userID string) (*umtypes.User, error) {
	u.log.WithField("id", userID).Info("get user info by id")
	resp, err := u.client.R().
		SetContext(ctx).
		SetResult(umtypes.User{}).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Get("/user/info/id/" + userID)
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, resp.Error().(*cherry.Err)
	}
	return resp.Result().(*umtypes.User), nil
}

func (u *httpUserManagerClient) UserLoginIDList(ctx context.Context) (map[string]string, error) {
	u.log.Info("get users list")
	resp, err := u.client.R().
		SetContext(ctx).
		SetResult(map[string]string{}).
		SetHeaders(httputil.RequestXHeadersMap(ctx)).
		Get("/user/loginid")
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, resp.Error().(*cherry.Err)
	}

	ret := resp.Result().(*map[string]string)

	return *ret, nil
}

type userManagerStub struct {
	log         *logrus.Entry
	givenLogins map[string]umtypes.User
}

func NewUserManagerStub() UserManagerClient {
	return &userManagerStub{
		log:         logrus.WithField("component", "user_manager_stub"),
		givenLogins: make(map[string]umtypes.User),
	}
}

func (u *userManagerStub) UserInfoByLogin(ctx context.Context, login string) (*umtypes.User, error) {
	u.log.WithField("id", login).Info("get user info by login")
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

func (u *userManagerStub) UserInfoByID(ctx context.Context, userID string) (*umtypes.User, error) {
	u.log.WithField("id", userID).Info("get user info by id")
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

func (u *userManagerStub) UserLoginIDList(ctx context.Context) (map[string]string, error) {
	u.log.Info("get user info by id")
	id := uuid.NewV4().String()
	return map[string]string{id: "fake-" + id + "@test.com"}, nil
}
