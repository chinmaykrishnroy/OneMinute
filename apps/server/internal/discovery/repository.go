package discovery

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Profile struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
}

func (r Repository) PersistConnection(ctx context.Context, match MatchState) (string, error) {
	users := []string{match.UserA, match.UserB}
	sort.Strings(users)
	shared := match.SharedInterests
	if shared == nil {
		shared = []string{}
	}
	startedAt := time.UnixMilli(match.StartedAt)
	endedAt := time.Now().UTC()
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO encounters(id,user_low,user_high,started_at,ended_at,intent,shared_interests,outcome)
		VALUES($1,$2,$3,$4,$5,$6,$7,'connected') ON CONFLICT(id) DO UPDATE SET outcome='connected'`, match.ID, users[0], users[1], startedAt, endedAt, match.Intent, shared); err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO connections(user_low,user_high,encounter_id) VALUES($1,$2,$3)
		ON CONFLICT(user_low,user_high) WHERE ended_at IS NULL DO UPDATE SET encounter_id=connections.encounter_id RETURNING id::text`, users[0], users[1], match.ID).Scan(&id)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return id, nil
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
