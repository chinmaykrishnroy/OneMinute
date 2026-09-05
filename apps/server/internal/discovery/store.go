package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	presenceTTL = 40 * time.Second
	matchTTL    = 5 * time.Minute
	recentTTL   = 15 * time.Minute
)

type queuedUser struct {
	ID           string
	ConnectionID string
	Preferences  Preferences
	JoinedAt     int64
}

type queuedState struct {
	ConnectionID string      `json:"connectionId"`
	Preferences  Preferences `json:"preferences"`
}

type matchState struct {
	ID, UserA, UserB, Intent string
	SharedInterests          []string
}

type Store struct {
	Redis  *redis.Client
	Prefix string
}

func (s Store) Connect(ctx context.Context, userID, connectionID string) error {
	return s.Redis.Set(ctx, s.presenceKey(userID), connectionID, presenceTTL).Err()
}

func (s Store) Heartbeat(ctx context.Context, userID, connectionID string) (bool, error) {
	result, err := s.Redis.Eval(ctx, heartbeatScript, []string{s.presenceKey(userID), s.matchKey(userID)}, connectionID, int(presenceTTL.Seconds()), int(matchTTL.Seconds()), s.prefix()).Int()
	return result == 1, err
}

func (s Store) Enqueue(ctx context.Context, userID, connectionID string, preferences Preferences, now time.Time) error {
	encoded, err := json.Marshal(queuedState{ConnectionID: connectionID, Preferences: preferences})
	if err != nil {
		return err
	}
	result, err := s.Redis.Eval(ctx, enqueueScript, []string{s.presenceKey(userID), s.matchKey(userID), s.key("queue"), s.key("preferences")}, userID, connectionID, string(encoded), now.UnixMilli()).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return errors.New("user is not available for queue")
	}
	return nil
}

func (s Store) LeaveQueue(ctx context.Context, userID, connectionID string) error {
	return s.Redis.Eval(ctx, leaveQueueScript, []string{s.presenceKey(userID), s.key("queue"), s.key("preferences")}, userID, connectionID).Err()
}

func (s Store) Candidates(ctx context.Context, userID string, limit int) ([]queuedUser, error) {
	items, err := s.Redis.ZRangeWithScores(ctx, s.key("queue"), 0, int64(limit)).Result()
	if err != nil {
		return nil, err
	}
	result := make([]queuedUser, 0, len(items))
	for _, item := range items {
		id, ok := item.Member.(string)
		if !ok || id == userID {
			continue
		}
		encoded, err := s.Redis.HGet(ctx, s.key("preferences"), id).Bytes()
		if err != nil {
			continue
		}
		var state queuedState
		if json.Unmarshal(encoded, &state) != nil || state.ConnectionID == "" {
			continue
		}
		activeConnection, err := s.Redis.Get(ctx, s.presenceKey(id)).Result()
		if err != nil || activeConnection != state.ConnectionID {
			_ = s.Redis.Eval(ctx, cleanupQueuedScript, []string{s.presenceKey(id), s.key("queue"), s.key("preferences")}, id, string(encoded), state.ConnectionID).Err()
			continue
		}
		result = append(result, queuedUser{ID: id, ConnectionID: state.ConnectionID, Preferences: state.Preferences, JoinedAt: int64(item.Score)})
	}
	return result, nil
}

func (s Store) Recent(ctx context.Context, userA, userB string) (bool, error) {
	count, err := s.Redis.Exists(ctx, s.recentKey(userA, userB)).Result()
	return count > 0, err
}

func (s Store) Claim(ctx context.Context, userA, userB, connectionA, connectionB, matchID, intent string, sharedInterests []string, eventA, eventB []byte, now time.Time) (bool, error) {
	shared, err := json.Marshal(sharedInterests)
	if err != nil {
		return false, err
	}
	result, err := s.Redis.Eval(ctx, claimScript, []string{
		s.presenceKey(userA), s.presenceKey(userB), s.matchKey(userA), s.matchKey(userB),
		s.key("match:") + matchID, s.recentKey(userA, userB), s.key("queue"), s.key("preferences"),
	}, userA, userB, connectionA, connectionB, matchID, now.UnixMilli(), int(matchTTL.Seconds()), int(recentTTL.Seconds()), intent, string(shared), string(eventA), string(eventB), s.prefix()).Int()
	return result == 1, err
}

func (s Store) CurrentMatch(ctx context.Context, userID string) (matchState, bool, error) {
	id, err := s.Redis.Get(ctx, s.matchKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return matchState{}, false, nil
	}
	if err != nil {
		return matchState{}, false, err
	}
	fields, err := s.Redis.HGetAll(ctx, s.key("match:")+id).Result()
	if err != nil || fields["a"] == "" || fields["b"] == "" {
		return matchState{}, false, err
	}
	var shared []string
	if err := json.Unmarshal([]byte(fields["sharedInterests"]), &shared); err != nil {
		return matchState{}, false, err
	}
	return matchState{ID: id, UserA: fields["a"], UserB: fields["b"], Intent: fields["intent"], SharedInterests: shared}, true, nil
}

func (s Store) Reconnect(ctx context.Context, userID, connectionID, matchID string) (bool, error) {
	result, err := s.Redis.Eval(ctx, reconnectScript, []string{s.presenceKey(userID), s.matchKey(userID), s.key("match:") + matchID}, userID, connectionID, matchID, s.prefix()).Int()
	return result == 1, err
}

func (s Store) Signal(ctx context.Context, userID, connectionID, matchID string, payload []byte) (bool, error) {
	result, err := s.Redis.Eval(ctx, signalScript, []string{s.presenceKey(userID), s.matchKey(userID), s.key("match:") + matchID}, userID, connectionID, matchID, string(payload), s.prefix()).Int()
	return result == 1, err
}

func (s Store) EndMatch(ctx context.Context, userID, connectionID, matchID, reason string) (bool, error) {
	result, err := s.Redis.Eval(ctx, endScript, []string{s.presenceKey(userID), s.matchKey(userID), s.key("match:") + matchID}, userID, connectionID, matchID, reason, s.prefix()).Int()
	return result == 1, err
}

func (s Store) Disconnect(ctx context.Context, userID, connectionID string, now time.Time) error {
	return s.Redis.Eval(ctx, disconnectScript, []string{s.presenceKey(userID), s.matchKey(userID), s.key("queue"), s.key("preferences")}, userID, connectionID, now.UnixMilli(), s.prefix()).Err()
}

func compatible(a, b Preferences) bool {
	intent := a.Intent == b.Intent || (a.Intent == "surprise_me" && b.Intent != "dating") || (b.Intent == "surprise_me" && a.Intent != "dating")
	if !intent {
		return false
	}
	for _, first := range a.Languages {
		for _, second := range b.Languages {
			if first == second {
				return true
			}
		}
	}
	return false
}

func sharedInterests(a, b []string) []string {
	set := map[string]bool{}
	for _, value := range a {
		set[value] = true
	}
	shared := []string{}
	for _, value := range b {
		if set[value] {
			shared = append(shared, value)
		}
	}
	sort.Strings(shared)
	return shared
}

func peer(match matchState, userID string) string {
	if match.UserA == userID {
		return match.UserB
	}
	return match.UserA
}

func (s Store) prefix() string {
	if s.Prefix != "" {
		return s.Prefix
	}
	return "discovery:"
}
func (s Store) key(suffix string) string         { return s.prefix() + suffix }
func (s Store) presenceKey(userID string) string { return s.key("presence:") + userID }
func (s Store) matchKey(userID string) string    { return s.key("user-match:") + userID }
func (s Store) channel(userID string) string     { return s.key("user:") + userID }
func (s Store) recentKey(a, b string) string {
	ids := []string{a, b}
	sort.Strings(ids)
	return s.key("recent:") + strings.Join(ids, ":")
}

const heartbeatScript = `
if redis.call('GET',KEYS[1])~=ARGV[1] then return 0 end
redis.call('EXPIRE',KEYS[1],ARGV[2])
local match=redis.call('GET',KEYS[2])
if match then
  redis.call('EXPIRE',KEYS[2],ARGV[3])
  redis.call('EXPIRE',ARGV[4]..'match:'..match,ARGV[3])
end
return 1`

const enqueueScript = `
if redis.call('GET',KEYS[1])~=ARGV[2] or redis.call('EXISTS',KEYS[2])==1 then return 0 end
redis.call('HSET',KEYS[4],ARGV[1],ARGV[3])
redis.call('ZADD',KEYS[3],ARGV[4],ARGV[1])
return 1`

const leaveQueueScript = `
if redis.call('GET',KEYS[1])~=ARGV[2] then return 0 end
redis.call('ZREM',KEYS[2],ARGV[1]);redis.call('HDEL',KEYS[3],ARGV[1]);return 1`

const cleanupQueuedScript = `
if redis.call('HGET',KEYS[3],ARGV[1])~=ARGV[2] then return 0 end
if redis.call('GET',KEYS[1])==ARGV[3] then return 0 end
redis.call('ZREM',KEYS[2],ARGV[1]);redis.call('HDEL',KEYS[3],ARGV[1]);return 1`

const reconnectScript = `
if redis.call('GET',KEYS[1])~=ARGV[2] or redis.call('GET',KEYS[2])~=ARGV[3] then return 0 end
local a=redis.call('HGET',KEYS[3],'a');local b=redis.call('HGET',KEYS[3],'b');local other
if ARGV[1]==a then other=b elseif ARGV[1]==b then other=a else return 0 end
redis.call('HDEL',KEYS[3],'disconnected:'..ARGV[1])
if other then redis.call('PUBLISH',ARGV[4]..'user:'..other,cjson.encode({version=1,type='peer.reconnected',matchId=ARGV[3],payload={}})) end
return 1`

const claimScript = `
if redis.call('GET',KEYS[1])~=ARGV[3] or redis.call('GET',KEYS[2])~=ARGV[4] then return 0 end
if redis.call('EXISTS',KEYS[3])==1 or redis.call('EXISTS',KEYS[4])==1 then return 0 end
if redis.call('ZSCORE',KEYS[7],ARGV[1])==false or redis.call('ZSCORE',KEYS[7],ARGV[2])==false then return 0 end
if redis.call('EXISTS',KEYS[6])==1 then return 0 end
redis.call('ZREM',KEYS[7],ARGV[1],ARGV[2])
redis.call('HDEL',KEYS[8],ARGV[1],ARGV[2])
redis.call('HSET',KEYS[5],'a',ARGV[1],'b',ARGV[2],'state','matched','createdAt',ARGV[6],'intent',ARGV[9],'sharedInterests',ARGV[10])
redis.call('EXPIRE',KEYS[5],ARGV[7])
redis.call('SET',KEYS[3],ARGV[5],'EX',ARGV[7])
redis.call('SET',KEYS[4],ARGV[5],'EX',ARGV[7])
redis.call('SET',KEYS[6],1,'EX',ARGV[8])
redis.call('PUBLISH',ARGV[13]..'user:'..ARGV[1],ARGV[11])
redis.call('PUBLISH',ARGV[13]..'user:'..ARGV[2],ARGV[12])
return 1`

const signalScript = `
if redis.call('GET',KEYS[1])~=ARGV[2] or redis.call('GET',KEYS[2])~=ARGV[3] then return 0 end
local a=redis.call('HGET',KEYS[3],'a');local b=redis.call('HGET',KEYS[3],'b')
local other
if ARGV[1]==a then other=b elseif ARGV[1]==b then other=a else return 0 end
if not other or redis.call('EXISTS',ARGV[5]..'presence:'..other)==0 then return 0 end
redis.call('PUBLISH',ARGV[5]..'user:'..other,ARGV[4]);return 1`

const endScript = `
if redis.call('GET',KEYS[1])~=ARGV[2] or redis.call('GET',KEYS[2])~=ARGV[3] then return 0 end
local a=redis.call('HGET',KEYS[3],'a');local b=redis.call('HGET',KEYS[3],'b')
if ARGV[1]~=a and ARGV[1]~=b then return 0 end
local message=cjson.encode({version=1,type='match.ended',matchId=ARGV[3],payload={reason=ARGV[4]}})
if a then redis.call('DEL',ARGV[5]..'user-match:'..a);redis.call('PUBLISH',ARGV[5]..'user:'..a,message) end
if b then redis.call('DEL',ARGV[5]..'user-match:'..b);redis.call('PUBLISH',ARGV[5]..'user:'..b,message) end
redis.call('DEL',KEYS[3]);return 1`

const disconnectScript = `
if redis.call('GET',KEYS[1])~=ARGV[2] then return 0 end
redis.call('DEL',KEYS[1]);redis.call('ZREM',KEYS[3],ARGV[1]);redis.call('HDEL',KEYS[4],ARGV[1])
local match=redis.call('GET',KEYS[2])
if not match then return 1 end
local matchKey=ARGV[4]..'match:'..match
local a=redis.call('HGET',matchKey,'a');local b=redis.call('HGET',matchKey,'b');local other
if ARGV[1]==a then other=b elseif ARGV[1]==b then other=a else return 1 end
redis.call('HSET',matchKey,'disconnected:'..ARGV[1],ARGV[3])
if other then redis.call('PUBLISH',ARGV[4]..'user:'..other,cjson.encode({version=1,type='peer.disconnected',matchId=match,payload={gracePlanned=true}})) end
return 1`
