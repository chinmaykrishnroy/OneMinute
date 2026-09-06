package communication

import "net/http"

func (h *Handler) conversations(w http.ResponseWriter, r *http.Request) {
	u, ok := h.user(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT c.id::text,p.user_id::text,p.display_name,p.avatar_url,
 COALESCE(last_message.body,''),COALESCE(last_message.created_at,c.created_at),
 (SELECT count(*) FROM notifications n WHERE n.user_id=$1 AND n.connection_id=c.id AND n.kind='message' AND n.read_at IS NULL)
 FROM connections c JOIN profiles p ON p.user_id=CASE WHEN c.user_low=$1 THEN c.user_high ELSE c.user_low END
 LEFT JOIN LATERAL(SELECT left(body,180) AS body,created_at FROM messages WHERE connection_id=c.id ORDER BY id DESC LIMIT 1) last_message ON true
 WHERE ($1=c.user_low OR $1=c.user_high) AND c.ended_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM blocks b WHERE (b.blocker_user_id=c.user_low AND b.blocked_user_id=c.user_high) OR (b.blocker_user_id=c.user_high AND b.blocked_user_id=c.user_low))
 AND NOT EXISTS(SELECT 1 FROM users WHERE id IN(c.user_low,c.user_high) AND status<>'active')
 ORDER BY COALESCE(last_message.created_at,c.created_at) DESC LIMIT 200`, u.ID)
	if err != nil {
		reply(w, 500, nil)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, person, name, avatar, preview string
		var updated any
		var unread int64
		if rows.Scan(&id, &person, &name, &avatar, &preview, &updated, &unread) != nil {
			reply(w, 500, nil)
			return
		}
		items = append(items, map[string]any{"id": id, "person": map[string]string{"id": person, "displayName": name, "avatarUrl": avatar}, "preview": preview, "updatedAt": updated, "unread": unread})
	}
	if rows.Err() != nil {
		reply(w, 500, nil)
		return
	}
	reply(w, 200, map[string]any{"connections": items})
}
