// Package artwork decides which anime an artwork sync walks: one season, a
// chosen list, or the whole catalogue page by page. Kept apart from the
// eventing package so it can be tested without the Kafka driver.
package artwork

import (
	"context"

	anime "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
)

// Source is the part of the anime repository the walk reads.
type Source interface {
	FindWithTheTVDBID(ctx context.Context, afterID string, limit int) ([]*anime.Anime, error)
	FindWithTheTVDBIDBySeason(ctx context.Context, season string) ([]*anime.Anime, error)
	FindByIDs(ctx context.Context, ids []string) ([]*anime.Anime, error)
}

// Selection says which anime to walk. IDs wins over Season; neither means
// the whole catalogue, paged by id from After.
type Selection struct {
	Season   string
	IDs      []string
	After    string
	PageSize int
}

// Targeted is true for a chosen set (season or ids), which is fetched once.
func (s Selection) Targeted() bool {
	return len(s.IDs) > 0 || s.Season != ""
}

// Walk returns a fetch function: each call yields the next batch, and an
// empty batch means the walk is over. A targeted selection is one batch; the
// catalogue walk pages by id and the caller advances After through Next.
type Walk struct {
	src     Source
	sel     Selection
	fetched bool
	after   string
}

func NewWalk(src Source, sel Selection) *Walk {
	if sel.PageSize <= 0 {
		sel.PageSize = 200
	}
	return &Walk{src: src, sel: sel, after: sel.After}
}

// Next fetches the next batch.
func (w *Walk) Next(ctx context.Context) ([]*anime.Anime, error) {
	if w.sel.Targeted() {
		if w.fetched {
			return nil, nil
		}
		w.fetched = true
		if len(w.sel.IDs) > 0 {
			return w.src.FindByIDs(ctx, w.sel.IDs)
		}
		return w.src.FindWithTheTVDBIDBySeason(ctx, w.sel.Season)
	}
	records, err := w.src.FindWithTheTVDBID(ctx, w.after, w.sel.PageSize)
	if err != nil {
		return nil, err
	}
	if len(records) > 0 {
		w.after = records[len(records)-1].ID
	}
	if len(records) < w.sel.PageSize {
		// Last page: the next call ends the walk without another query.
		w.fetched = true
	}
	if w.fetched && len(records) == 0 {
		return nil, nil
	}
	return records, nil
}

// Done reports whether the catalogue walk has reached its last page.
func (w *Walk) Done() bool { return w.fetched }
