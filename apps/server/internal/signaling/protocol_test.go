package signaling

import "testing"

func TestProtocolValidation(t *testing.T) {
	for _, body := range []string{
		`{"version":2,"type":"presence.heartbeat","payload":{}}`,
		`{"version":1,"type":"send.to.user","payload":{}}`,
		`{"version":1,"type":"webrtc.offer","payload":{"type":"answer","sdp":"x"}}`,
		`{"version":1,"type":"presence.heartbeat","payload":null}`,
		`{"version":1,"type":"presence.heartbeat","payload":{}}{}`,
		`{"version":1,"type":"presence.heartbeat","payload":{},"to":"someone"}`,
	} {
		if _, err := Decode([]byte(body)); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
	for _, body := range []string{
		`{"version":1,"type":"presence.heartbeat","payload":{}}`,
		`{"version":1,"type":"webrtc.ice","payload":{"candidate":"candidate:x","sdpMid":"0","sdpMLineIndex":0,"usernameFragment":"x"}}`,
	} {
		if _, err := Decode([]byte(body)); err != nil {
			t.Errorf("rejected valid event: %v", err)
		}
	}
}
