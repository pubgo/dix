package mongocmd

import (
	"context"
	"github.com/aginetwork7/portal-server/internal/logic/assettrackinglogic"
	"github.com/pubgo/dix"
	"github.com/urfave/cli/v3"

	"github.com/aginetwork7/common/pkg/component/gpt"
	"github.com/aginetwork7/common/pkg/component/openai"

	commongo "github.com/aginetwork7/portal-server/internal/component/mongo"
	"github.com/aginetwork7/portal-server/internal/component/s3cli"
	logicevent "github.com/aginetwork7/portal-server/internal/logic/event"
)

type params struct {
	Gpt                *gpt.Client
	Mongo              *commongo.Client
	OpenAi             *openai.Client
	S3                 *s3cli.Client
	EventCfg           *logicevent.Config
	AssetTrackingLogic *assettrackinglogic.AssetTrackingLogic
}

func New(di *dix.Dix) *cli.Command {
	p := &params{}
	return &cli.Command{
		Name:  "mongo",
		Usage: "mongo cmd",
		Before: func(ctx context.Context, command *cli.Command) error {
			p = dix.Inject(di, p)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:        "migrate_asset_tracking_s3",
				Description: "copy temp s3",
				Action: func(ctx context.Context, command *cli.Command) error {
					p.AssetTrackingLogic.mi
					return nil
				},
			},
		},
	}
}
