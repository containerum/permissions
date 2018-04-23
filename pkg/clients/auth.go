package clients

import (
	"context"
	"fmt"
	"io"
	"time"

	"git.containerum.net/ch/auth/proto"
	"git.containerum.net/ch/kube-client/pkg/cherry/adaptors/cherrygrpc"
	"git.containerum.net/ch/kube-client/pkg/cherry/resource-service"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// AuthSvc is an interface to auth service
type AuthSvc interface {
	UpdateUserAccess(ctx context.Context, userID string, access *authProto.ResourcesAccess) error

	// for connections closing
	io.Closer
}

type authSvcGRPC struct {
	client authProto.AuthClient
	addr   string
	log    *logrus.Entry
	conn   *grpc.ClientConn
}

// NewAuthSvcGRPC creates grpc client to auth service. It does nothing but logs actions.
func NewAuthSvcGRPC(addr string) (as AuthSvc, err error) {
	ret := authSvcGRPC{
		log:  logrus.WithField("component", "auth_client"),
		addr: addr,
	}

	cherrygrpc.JSONMarshal = jsoniter.ConfigFastest.Marshal
	cherrygrpc.JSONUnmarshal = jsoniter.ConfigFastest.Unmarshal

	ret.log.Debugf("grpc connect to %s", addr)
	ret.conn, err = grpc.Dial(addr,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			cherrygrpc.UnaryClientInterceptor(rserrors.ErrInternal),
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

	return ret, nil
}

func (as authSvcGRPC) UpdateUserAccess(ctx context.Context, userID string, access *authProto.ResourcesAccess) error {
	as.log.WithField("user_id", userID).Infoln("update user access")
	_, err := as.client.UpdateAccess(ctx, &authProto.UpdateAccessRequest{
		Users: []*authProto.UpdateAccessRequestElement{
			{UserId: userID, Access: access},
		},
	})
	return err
}

func (as authSvcGRPC) String() string {
	return fmt.Sprintf("auth grpc client: addr=%v", as.addr)
}

func (as authSvcGRPC) Close() error {
	return as.conn.Close()
}

type authSvcDummy struct {
	log *logrus.Entry
}

// NewDummyAuthSvc creates dummy auth client
func NewDummyAuthSvc() AuthSvc {
	return authSvcDummy{
		log: logrus.WithField("component", "auth_stub"),
	}
}

func (as authSvcDummy) UpdateUserAccess(ctx context.Context, userID string, access *authProto.ResourcesAccess) error {
	as.log.WithField("user_id", userID).Infoln("update user access to %+v", access)
	return nil
}

func (authSvcDummy) String() string {
	return "ch-auth client dummy"
}

func (authSvcDummy) Close() error {
	return nil
}
