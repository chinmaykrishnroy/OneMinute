package discovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
)

const maxFrameBytes = 64 << 10

var languagePattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[A-Z]{2})?$`)

var allowedIntents = map[string]bool{
	"surprise_me": true, "new_friends": true, "dating": true,
	"gaming": true, "language_exchange": true, "tech_ideas": true,
	"professional_networking": true,
}

var allowedInterests = map[string]bool{
	"ai": true, "art": true, "books": true, "films": true,
	"fitness": true, "gaming": true, "music": true, "nature": true,
	"photography": true, "science": true, "technology": true, "travel": true,
}

type Preferences struct {
	Intent    string   `json:"intent"`
	Languages []string `json:"languages"`
	Interests []string `json:"interests"`
}

type Envelope struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	MatchID   string          `json:"matchId,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

func event(kind, matchID string, payload any) Envelope {
	encoded, _ := json.Marshal(payload)
	return Envelope{Version: 1, Type: kind, MatchID: matchID, Payload: encoded}
}

func decode(data []byte) (Envelope, error) {
	var envelope Envelope
	if len(data) > maxFrameBytes || strict(data, &envelope) != nil || envelope.Version != 1 || len(envelope.RequestID) > 128 {
		return envelope, errors.New("invalid envelope")
	}
	switch envelope.Type {
	case "queue.join":
		var preferences Preferences
		if strict(envelope.Payload, &preferences) != nil {
			return envelope, errors.New("invalid preferences")
		}
		normalized, err := normalizePreferences(preferences)
		if err != nil {
			return envelope, err
		}
		envelope.Payload, _ = json.Marshal(normalized)
	case "queue.leave", "presence.heartbeat", "match.leave":
		var empty struct{}
		if strict(envelope.Payload, &empty) != nil {
			return envelope, errors.New("invalid empty payload")
		}
	case "webrtc.offer", "webrtc.answer":
		var session struct {
			Type string `json:"type"`
			SDP  string `json:"sdp"`
		}
		if strict(envelope.Payload, &session) != nil || session.Type != strings.TrimPrefix(envelope.Type, "webrtc.") || len(session.SDP) == 0 || len(session.SDP) > 60000 {
			return envelope, errors.New("invalid SDP")
		}
	case "webrtc.ice":
		var candidate struct {
			Candidate        string  `json:"candidate"`
			SDPMid           *string `json:"sdpMid"`
			SDPMLineIndex    *uint16 `json:"sdpMLineIndex"`
			UsernameFragment *string `json:"usernameFragment"`
		}
		if strict(envelope.Payload, &candidate) != nil || len(candidate.Candidate) > 4096 || (candidate.SDPMid != nil && len(*candidate.SDPMid) > 256) || (candidate.UsernameFragment != nil && len(*candidate.UsernameFragment) > 256) {
			return envelope, errors.New("invalid ICE candidate")
		}
	default:
		return envelope, errors.New("unknown event type")
	}
	return envelope, nil
}

func normalizePreferences(value Preferences) (Preferences, error) {
	value.Intent = strings.TrimSpace(strings.ToLower(value.Intent))
	if !allowedIntents[value.Intent] || len(value.Languages) == 0 || len(value.Languages) > 3 || len(value.Interests) > 8 {
		return Preferences{}, errors.New("invalid preferences")
	}
	var ok bool
	value.Languages, ok = unique(value.Languages, func(item string) (string, bool) {
		item = strings.TrimSpace(item)
		return item, languagePattern.MatchString(item)
	})
	if !ok {
		return Preferences{}, errors.New("invalid languages")
	}
	value.Interests, ok = unique(value.Interests, func(item string) (string, bool) {
		item = strings.TrimSpace(strings.ToLower(item))
		return item, allowedInterests[item]
	})
	if !ok {
		return Preferences{}, errors.New("invalid interests")
	}
	return value, nil
}

func unique(values []string, normalize func(string) (string, bool)) ([]string, bool) {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		item, ok := normalize(value)
		if !ok {
			return nil, false
		}
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	sort.Strings(result)
	return result, true
}

func strict(data []byte, target any) error {
	if len(data) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("object required")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}
