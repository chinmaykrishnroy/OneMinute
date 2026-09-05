package signaling

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

type Envelope struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	MatchID   string          `json:"matchId,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

var roomPattern = regexp.MustCompile("^[a-f0-9]{32}$")

func ValidRoom(id string) bool { return roomPattern.MatchString(id) }
func Event(kind, matchID string, payload any) Envelope {
	b, _ := json.Marshal(payload)
	return Envelope{Version: 1, Type: kind, MatchID: matchID, Payload: b}
}
func Decode(data []byte) (Envelope, error) {
	var e Envelope
	if len(data) > 64<<10 {
		return e, errors.New("message too large")
	}
	if err := strict(data, &e); err != nil {
		return e, err
	}
	if e.Version != 1 || len(e.RequestID) > 128 || (e.MatchID != "" && !ValidRoom(e.MatchID)) {
		return e, errors.New("invalid envelope")
	}
	switch e.Type {
	case "room.join":
		var p struct {
			RoomID string `json:"roomId"`
		}
		if err := strict(e.Payload, &p); err != nil || !ValidRoom(p.RoomID) {
			return e, errors.New("invalid room")
		}
	case "webrtc.offer", "webrtc.answer":
		var p struct {
			Type string `json:"type"`
			SDP  string `json:"sdp"`
		}
		if err := strict(e.Payload, &p); err != nil || p.Type != e.Type[7:] || len(p.SDP) == 0 || len(p.SDP) > 60000 {
			return e, errors.New("invalid SDP")
		}
	case "webrtc.ice":
		var p struct {
			Candidate        string  `json:"candidate"`
			SDPMid           *string `json:"sdpMid"`
			SDPMLineIndex    *uint16 `json:"sdpMLineIndex"`
			UsernameFragment *string `json:"usernameFragment"`
		}
		if err := strict(e.Payload, &p); err != nil || len(p.Candidate) > 4096 || (p.SDPMid != nil && len(*p.SDPMid) > 256) || (p.UsernameFragment != nil && len(*p.UsernameFragment) > 256) {
			return e, errors.New("invalid candidate")
		}
	case "presence.heartbeat", "match.leave":
		var p struct{}
		if err := strict(e.Payload, &p); err != nil {
			return e, err
		}
	default:
		return e, errors.New("unknown event type")
	}
	return e, nil
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
