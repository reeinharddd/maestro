package skill

import (
	"sort"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
)

// Estimator calculates context usage estimates from stored skill metadata.
type Estimator struct {
	db db.DBInterface
}

// NewEstimator creates an Estimator backed by the given database.
func NewEstimator(database db.DBInterface) *Estimator {
	return &Estimator{db: database}
}

// Estimate returns a ContextEstimate summarizing size by source, category, and heaviest skills.
func (e *Estimator) Estimate() (*models.ContextEstimate, error) {
	skills, err := e.db.ListSkills()
	if err != nil {
		return nil, err
	}

	est := &models.ContextEstimate{
		TotalBytes:  0,
		TotalSkills: len(skills),
		BySource:    make(map[string]int64),
		ByCategory:  make(map[string]int64),
	}

	for _, s := range skills {
		est.TotalBytes += s.SizeBytes
		if s.Source != "" {
			est.BySource[s.Source] += s.SizeBytes
		}
		if s.Category != "" {
			est.ByCategory[s.Category] += s.SizeBytes
		}
	}

	// Build sorted list by size descending
	type sizedEntry struct {
		id    string
		bytes int64
	}
	var sorted []sizedEntry
	for _, s := range skills {
		if s.SizeBytes > 0 {
			sorted = append(sorted, sizedEntry{s.ID, s.SizeBytes})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].bytes > sorted[j].bytes
	})

	// Top 10 heaviest skills
	limit := 10
	if len(sorted) < limit {
		limit = len(sorted)
	}
	est.Heaviest = make([]models.SkillSizeEntry, 0, limit)
	for i := 0; i < limit; i++ {
		// Find the skill for source/category info
		for _, s := range skills {
			if s.ID == sorted[i].id {
				est.Heaviest = append(est.Heaviest, models.SkillSizeEntry{
					ID:        s.ID,
					Source:    s.Source,
					Category:  s.Category,
					SizeBytes: s.SizeBytes,
				})
				break
			}
		}
	}

	return est, nil
}
