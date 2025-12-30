package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type Create struct {
	dao *dao.FileDrinkDAO
}

func NewCreate(dao *dao.FileDrinkDAO) *Create {
	return &Create{dao: dao}
}

func (c *Create) Execute(ctx *middleware.Context, name string) (models.Drink, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.Drink{}, fmt.Errorf("name is required")
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Drink{}, fmt.Errorf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Drink{}, err
	}

	id, err := newID()
	if err != nil {
		return models.Drink{}, err
	}

	record := dao.Drink{
		ID:   id,
		Name: name,
	}

	if err := c.dao.Add(ctx, record); err != nil {
		return models.Drink{}, err
	}

	ctx.AddEvent(events.DrinkCreated{DrinkID: id, Name: name})

	return record.ToDomain(), nil
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
