package discovery

import (
	"encoding/json"
	"testing"
)

func TestPreferencesAndCompatibility(t *testing.T) {
	message := []byte(`{"version":1,"type":"queue.join","payload":{"intent":"New_Friends","languages":["en-US","hi"],"interests":["Music","ai","music"]}}`)
	envelope, err := decode(message)
	if err != nil {
		t.Fatal(err)
	}
	var got Preferences
	if err := json.Unmarshal(envelope.Payload, &got); err != nil {
		t.Fatal(err)
	}
	if got.Intent != "new_friends" || len(got.Interests) != 2 || got.Interests[0] != "ai" {
		t.Fatalf("unexpected normalization: %+v", got)
	}
	if !compatible(got, Preferences{Intent: "surprise_me", Languages: []string{"hi"}}) {
		t.Fatal("compatible surprise-me pair rejected")
	}
	if compatible(got, Preferences{Intent: "dating", Languages: []string{"hi"}}) {
		t.Fatal("dating matched without mutual dating intent")
	}
	if compatible(got, Preferences{Intent: "new_friends", Languages: []string{"fr"}}) {
		t.Fatal("pair without a shared language accepted")
	}
}

func TestRejectsInvalidDiscoveryEvents(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`{"version":1,"type":"queue.join","payload":{"intent":"dating","languages":[],"interests":[]}}`),
		[]byte(`{"version":1,"type":"queue.join","payload":{"intent":"dating","languages":["en"],"interests":["unknown"]}}`),
		[]byte(`{"version":1,"type":"queue.join","payload":{"intent":"surprise_me","languages":["en"],"interests":[],"userId":"forged"}}`),
		[]byte(`{"version":1,"type":"unknown","payload":{}}`),
	} {
		if _, err := decode(data); err == nil {
			t.Fatalf("accepted invalid event: %s", data)
		}
	}
}
