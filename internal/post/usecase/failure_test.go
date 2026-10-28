package usecase

import (
	"aphrodite/internal/post/domain"
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"
)

type failureRepo struct {
	*updatePostRepo
	findErr, writeErr error
	writes            int
}

func (r *failureRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.updatePostRepo.FindByID(ctx, id)
}
func (r *failureRepo) Update(ctx context.Context, p *domain.Post) error {
	r.writes++
	if r.writeErr != nil {
		return r.writeErr
	}
	return r.updatePostRepo.Update(ctx, p)
}
func (r *failureRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.writes++
	if r.writeErr != nil {
		return r.writeErr
	}
	return r.updatePostRepo.Delete(ctx, id)
}
func TestWrites_PropagateRepositoryFailures(t *testing.T) {
	failure := errors.New("database unavailable")
	for _, tc := range []struct {
		name              string
		findErr, writeErr error
	}{
		{"lookup missing", domain.ErrNotFound, nil}, {"lookup unavailable", failure, nil}, {"concurrent deletion", nil, domain.ErrNotFound}, {"write unavailable", nil, failure},
	} {
		for _, operation := range []string{"update", "delete"} {
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				id, author := uuid.New(), uuid.New()
				repo := &failureRepo{updatePostRepo: newUpdatePostRepo(&domain.Post{ID: id, AuthorID: author, Content: "old", Title: "old"}), findErr: tc.findErr, writeErr: tc.writeErr}
				var err error
				if operation == "update" {
					_, err = NewUpdatePost(repo, UpdatePostConfig{}, nil).Execute(context.Background(), UpdateInput{PostID: id, CallerID: author, Content: "new", Title: "new"})
				} else {
					err = NewDeletePost(repo).Execute(context.Background(), DeleteInput{PostID: id, CallerID: author})
				}
				want := tc.findErr
				if want == nil {
					want = tc.writeErr
				}
				if !errors.Is(err, want) {
					t.Fatalf("error=%v, want %v", err, want)
				}
				if tc.findErr != nil && repo.writes != 0 {
					t.Fatal("write attempted after failed lookup")
				}
			})
		}
	}
}
