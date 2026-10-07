package artwork

import (
	"context"
	"testing"

	anime "github.com/weeb-vip/thetvdb-enrichment/internal/db/repositories/anime"
)

type fakeSource struct {
	bySeason, byIDs, paged int
	catalogue              []*anime.Anime
}

func rec(id string) *anime.Anime { return &anime.Anime{ID: id} }

func (f *fakeSource) FindWithTheTVDBID(_ context.Context, after string, limit int) ([]*anime.Anime, error) {
	f.paged++
	var out []*anime.Anime
	for _, a := range f.catalogue {
		if a.ID > after {
			out = append(out, a)
		}
		if len(out) == limit {
			break
		}
	}
	return out, nil
}
func (f *fakeSource) FindWithTheTVDBIDBySeason(_ context.Context, season string) ([]*anime.Anime, error) {
	f.bySeason++
	return []*anime.Anime{rec("s1"), rec("s2")}, nil
}
func (f *fakeSource) FindByIDs(_ context.Context, ids []string) ([]*anime.Anime, error) {
	f.byIDs++
	var out []*anime.Anime
	for _, id := range ids {
		out = append(out, rec(id))
	}
	return out, nil
}

func drain(t *testing.T, w *Walk) []string {
	t.Helper()
	var ids []string
	for i := 0; i < 100; i++ {
		batch, err := w.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(batch) == 0 {
			return ids
		}
		for _, a := range batch {
			ids = append(ids, a.ID)
		}
	}
	t.Fatal("the walk never ended")
	return nil
}

// The regression this exists for: 1.10.1 took --season and walked the whole
// catalogue anyway, re-pulling 1,100 banners before it was stopped.
func TestASeasonWalksOnlyThatSeasonAndNeverTheCatalogue(t *testing.T) {
	src := &fakeSource{catalogue: []*anime.Anime{rec("a"), rec("b"), rec("c")}}
	ids := drain(t, NewWalk(src, Selection{Season: "FALL_2026"}))
	if len(ids) != 2 || ids[0] != "s1" || src.bySeason != 1 || src.paged != 0 {
		t.Fatalf("ids %v, bySeason %d, paged %d", ids, src.bySeason, src.paged)
	}
}

func TestAnIDListWalksExactlyThoseAnime(t *testing.T) {
	src := &fakeSource{catalogue: []*anime.Anime{rec("a"), rec("b")}}
	ids := drain(t, NewWalk(src, Selection{IDs: []string{"x", "y"}, Season: "ignored"}))
	if len(ids) != 2 || ids[0] != "x" || src.byIDs != 1 || src.bySeason != 0 || src.paged != 0 {
		t.Fatalf("ids %v, byIDs %d, bySeason %d, paged %d", ids, src.byIDs, src.bySeason, src.paged)
	}
}

func TestTheCatalogueWalkPagesByIDFromAfter(t *testing.T) {
	src := &fakeSource{catalogue: []*anime.Anime{rec("a"), rec("b"), rec("c"), rec("d"), rec("e")}}
	ids := drain(t, NewWalk(src, Selection{After: "a", PageSize: 2}))
	if len(ids) != 4 || ids[0] != "b" || ids[3] != "e" {
		t.Fatalf("ids %v", ids)
	}
	// b,c | d,e | empty: two full pages, and the empty one that ends it.
	if src.paged != 3 {
		t.Fatalf("paged %d queries", src.paged)
	}
}
