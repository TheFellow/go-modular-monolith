package drinks

import (
	"context"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/commands"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type Module struct {
	queries *queries.Queries
	create  *commands.Create
}

func NewModule(drinksDataPath string) (*Module, error) {
	d := dao.NewFileDrinkDAO(drinksDataPath)
	if err := d.Load(context.Background()); err != nil {
		return nil, err
	}

	return &Module{
		queries: queries.NewWithDAO(d),
		create:  commands.NewCreate(d),
	}, nil
}

func (m *Module) List(ctx context.Context, req ListRequest) (ListResponse, error) {
	return middleware.RunQuery(ctx, drinksauthz.ActionList, func(mctx *middleware.Context, _ ListRequest) (ListResponse, error) {
		ds, err := m.queries.List(mctx)
		if err != nil {
			return ListResponse{}, err
		}
		return ListResponse{Drinks: ds}, nil
	}, req)
}

func (m *Module) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	return middleware.RunQuery(ctx, drinksauthz.ActionGet, func(mctx *middleware.Context, req GetRequest) (GetResponse, error) {
		d, err := m.queries.Get(mctx, req.ID)
		if err != nil {
			return GetResponse{}, err
		}
		return GetResponse{Drink: d}, nil
	}, req)
}

func (m *Module) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	resource := cedar.Entity{
		UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drinks::Catalog"), cedar.String("default")),
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}

	return middleware.RunCommand(ctx, drinksauthz.ActionCreate, resource, func(mctx *middleware.Context, req CreateRequest) (CreateResponse, error) {
		d, err := m.create.Execute(mctx, req.Name)
		if err != nil {
			return CreateResponse{}, err
		}
		return CreateResponse{Drink: d}, nil
	}, req)
}
