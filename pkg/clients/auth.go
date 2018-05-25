package clients

import (
	"context"
	"fmt"
	"io"
	"time"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/containerum/cherry/adaptors/cherrygrpc"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// AuthClient is an interface to auth service
type AuthClient interface {
	UpdateUserAccess(ctx context.Context, userID string, access *authProto.ResourcesAccess) error

	// for connections closing
	io.Closer
}

type AuthGRPCClient struct {
	client authProto.AuthClient
	addr   string
	log    *logrus.Entry
	conn   *grpc.ClientConn
}

// NewAuthGRPCClient creates grpc client to auth service. It does nothing but logs actions.
func NewAuthGRPCClient(addr string) (as *AuthGRPCClient, err error) {
	ret := AuthGRPCClient{
		log:  logrus.WithField("component", "auth_client"),
		addr: addr,
	}

	cherrygrpc.JSONMarshal = jsoniter.ConfigFastest.Marshal
	cherrygrpc.JSONUnmarshal = jsoniter.ConfigFastest.Unmarshal

	ret.log.Debugf("grpc connect to %s", addr)
	ret.conn, err = grpc.Dial(addr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			cherrygrpc.UnaryClientInterceptor(errors.ErrInternal),
			grpc_logrus.UnaryClientInterceptor(ret.log),
		)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                5 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}))
	if err != nil {
		return
	}
	ret.client = authProto.NewAuthClient(ret.conn)

	return &ret, nil
}

func (as AuthGRPCClient) UpdateUserAccess(ctx context.Context, userID string, access *authProto.ResourcesAccess) error {
	as.log.WithField("user_id", userID).Debugf("update user access to %+v", access)
	_, err := as.client.UpdateAccess(ctx, &authProto.UpdateAccessRequest{
		Users: []*authProto.UpdateAccessRequestElement{
			{UserId: userID, Access: access},
		},
	})
	return err
}

func (as AuthGRPCClient) String() string {
	return fmt.Sprintf("auth grpc client: addr=%v", as.addr)
}

func (as AuthGRPCClient) Close() error {
	return as.conn.Close()
}

type AuthDummyClient struct {
	log *logrus.Entry
}

// NewAuthDummyClient creates dummy auth client
func NewAuthDummyClient() AuthClient {
	return AuthDummyClient{
		log: logrus.WithField("component", "auth_stub"),
	}
}

func (as AuthDummyClient) UpdateUserAccess(ctx context.Context, userID string, access *authProto.ResourcesAccess) error {
	as.log.WithField("user_id", userID).Debugf("update user access to %+v", access)
	return nil
}

func (AuthDummyClient) String() string {
	return "auth dummy client"
}

func (AuthDummyClient) Close() error {
	return nil
}
