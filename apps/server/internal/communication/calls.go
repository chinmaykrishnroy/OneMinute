package communication

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type Call struct {
	ID           string `json:"id"`
	ConnectionID string `json:"connectionId"`
	Caller       string `json:"caller"`
	Callee       string `json:"callee"`
	Name         string `json:"name"`
	Video        bool   `json:"video"`
}

func (h *Handler) invite(w http.ResponseWriter, r *http.Request) {
	u, p, ok := h.authorized(w, r)
	if !ok {
		return
	}
	if !h.allow(w, r, u.ID, "invites", 6) {
		return
	}
	var b struct {
		Video bool `json:"video"`
	}
	if decode(w, r, &b) != nil {
		reply(w, 400, nil)
		return
	}
	call := Call{uuid.NewString(), r.PathValue("id"), u.ID, p, u.DisplayName, b.Video}
	if h.DB.QueryRow(r.Context(), "SELECT display_name FROM profiles WHERE user_id=$1", u.ID).Scan(&call.Name) != nil {
		reply(w, 503, nil)
		return
	}
	data, _ := json.Marshal(call)
	ctx := r.Context()
	result, e := h.Redis.Eval(ctx, `if redis.call('EXISTS',KEYS[1])==1 or redis.call('EXISTS',KEYS[2])==1 then return 0 end
redis.call('SET',KEYS[1],ARGV[1],'EX',50);redis.call('SET',KEYS[2],ARGV[1],'EX',50);redis.call('HSET',KEYS[3],'data',ARGV[2],'state','ringing');redis.call('EXPIRE',KEYS[3],45);return 1`, []string{"communication:busy:" + u.ID, "communication:busy:" + p, "communication:call:" + call.ID}, call.ID, string(data)).Int()
	if e != nil || result != 1 {
		reply(w, 409, map[string]string{"error": "One of you is already in a call. Try again shortly."})
		return
	}
	_, e = h.DB.Exec(ctx, `INSERT INTO notifications(user_id,connection_id,kind,reference) VALUES($1,$2,'call',$3)`, p, call.ConnectionID, call.ID)
	if e != nil {
		h.Redis.Del(ctx, "communication:call:"+call.ID, "communication:busy:"+u.ID, "communication:busy:"+p)
		reply(w, 500, nil)
		return
	}
	h.emit(ctx, p, map[string]any{"type": "call.invited", "call": call})
	reply(w, 200, call)
}
func (h *Handler) callAction(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	if _, e := uuid.Parse(r.PathValue("id")); e != nil {
		reply(w, 400, nil)
		return
	}
	ctx := r.Context()
	key := "communication:call:" + r.PathValue("id")
	fields, e := h.Redis.HGetAll(ctx, key).Result()
	var call Call
	if e != nil || json.Unmarshal([]byte(fields["data"]), &call) != nil || (call.Caller != u.ID && call.Callee != u.ID) {
		reply(w, 404, nil)
		return
	}
	target := call.Callee
	if u.ID == target {
		target = call.Caller
	}
	var b struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if decode(w, r, &b) != nil {
		reply(w, 400, nil)
		return
	}
	if _, e = h.peer(ctx, u.ID, call.ConnectionID); e != nil {
		b.Type = "end"
	}
	switch b.Type {
	case "accept":
		if u.ID != call.Callee {
			reply(w, 403, nil)
			return
		}
		v, e := h.Redis.Eval(ctx, `if redis.call('HGET',KEYS[1],'state')~='ringing' then return 0 end;redis.call('HSET',KEYS[1],'state','active');redis.call('EXPIRE',KEYS[1],75);redis.call('EXPIRE',KEYS[2],80);redis.call('EXPIRE',KEYS[3],80);return 1`, []string{key, "communication:busy:" + call.Caller, "communication:busy:" + call.Callee}).Int()
		if e != nil || v != 1 {
			reply(w, 409, nil)
			return
		}
	case "end", "decline":
		_, _ = h.Redis.Eval(ctx, `redis.call('DEL',KEYS[1]);for i=2,3 do if redis.call('GET',KEYS[i])==ARGV[1] then redis.call('DEL',KEYS[i]) end end;return 1`, []string{key, "communication:busy:" + call.Caller, "communication:busy:" + call.Callee}, call.ID).Result()
	case "heartbeat":
		if fields["state"] != "active" {
			reply(w, 409, nil)
			return
		}
		_, e = h.Redis.Eval(ctx, `if redis.call('HGET',KEYS[1],'state')~='active' then return 0 end;redis.call('EXPIRE',KEYS[1],75);for i=2,3 do if redis.call('GET',KEYS[i])==ARGV[1] then redis.call('EXPIRE',KEYS[i],80) else redis.call('SET',KEYS[i],ARGV[1],'EX',80,'NX') end end;return 1`, []string{key, "communication:busy:" + call.Caller, "communication:busy:" + call.Callee}, call.ID).Result()
		if e != nil {
			reply(w, 503, nil)
			return
		}
		reply(w, 200, nil)
		return
	case "offer", "answer":
		if fields["state"] != "active" || (b.Type == "offer" && u.ID != call.Caller) || (b.Type == "answer" && u.ID != call.Callee) {
			reply(w, 403, nil)
			return
		}
		var s struct {
			Type string `json:"type"`
			SDP  string `json:"sdp"`
		}
		if json.Unmarshal(b.Payload, &s) != nil || s.Type != b.Type || len(s.SDP) < 1 || len(s.SDP) > 60000 {
			reply(w, 400, nil)
			return
		}
	case "ice":
		if fields["state"] != "active" || len(b.Payload) > 6000 {
			reply(w, 400, nil)
			return
		}
		var s struct {
			Candidate string `json:"candidate"`
		}
		if json.Unmarshal(b.Payload, &s) != nil {
			reply(w, 400, nil)
			return
		}
	case "media":
		if fields["state"] != "active" {
			reply(w, 403, nil)
			return
		}
		var s struct {
			Video bool `json:"video"`
			Audio bool `json:"audio"`
		}
		if json.Unmarshal(b.Payload, &s) != nil {
			reply(w, 400, nil)
			return
		}
	default:
		reply(w, 400, nil)
		return
	}
	h.emit(ctx, target, map[string]any{"type": "call." + b.Type, "call": call, "payload": b.Payload, "at": time.Now().UnixMilli()})
	if b.Type == "accept" || b.Type == "end" || b.Type == "decline" {
		h.emit(ctx, u.ID, map[string]any{"type": "call." + b.Type, "call": call})
	}
	reply(w, 200, map[string]bool{"ok": true})
}
