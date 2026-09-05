package discovery

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Profile struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
}

type Repository struct{ DB *pgxpool.Pool }

func (r Repository) EligiblePair(ctx context.Context, userA, userB string) (bool, error) {
	var eligible bool
	err := r.DB.QueryRow(ctx, `
		SELECT count(*) = 2
		FROM users
		WHERE id = ANY($1::uuid[]) AND status = 'active'
		AND NOT EXISTS (
			SELECT 1 FROM blocks
			WHERE (blocker_user_id=$2 AND blocked_user_id=$3)
			   OR (blocker_user_id=$3 AND blocked_user_id=$2)
		)`, []string{userA, userB}, userA, userB).Scan(&eligible)
	return eligible, err
}

func (r Repository) Profile(ctx context.Context, userID string) (Profile, error) {
	var profile Profile
	err := r.DB.QueryRow(ctx, `SELECT u.id::text,p.display_name,p.avatar_url
		FROM users u JOIN profiles p ON p.user_id=u.id
		WHERE u.id=$1 AND u.status='active'`, userID).Scan(&profile.ID, &profile.DisplayName, &profile.AvatarURL)
	return profile, err
}
