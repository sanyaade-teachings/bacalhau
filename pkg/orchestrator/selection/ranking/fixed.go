package ranking

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// node Ranker that always returns the same set of nodes
type fixedRanker struct {
	ranks []int
}

func NewFixedRanker(ranks ...int) *fixedRanker {
	return &fixedRanker{
		ranks: ranks,
	}
}

func (f *fixedRanker) RankNodes(_ context.Context, _ models.Job, _ time.Duration, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))
	for i, rank := range f.ranks {
		ranks[i] = orchestrator.NodeRank{
			NodeInfo: nodes[i],
			Rank:     rank,
		}
	}
	return ranks, nil
}
