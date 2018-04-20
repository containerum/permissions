package server

import (
	"context"

	"git.containerum.net/ch/auth/proto"
)

type AccessActions interface {
	GetUserAccesses(ctx context.Context, userID string) ([]*authProto.AccessObject, error)
}
