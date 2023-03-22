package onboarding

import (
	"context"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
)

//go:generate ../../../bin/mockery -name=inventoryService -case=snake -inpkg -testonly

type inventoryService interface {
	List(ctx context.Context, filters models.ServiceFilters) ([]inventorypb.Service, error)
}
