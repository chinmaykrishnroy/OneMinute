// rtc-probe is an optional Pion headless peer for the development networking room.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type envelope struct {
	Version int             `json:"version"`
	Type    string          `json:"type"`
	MatchID string          `json:"matchId,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

func main() {
	api := flag.String("api", "http://localhost:8080", "development Go API")
	origin := flag.String("origin", "http://localhost:3000", "configured web origin")
	room := flag.String("room", "", "existing room; omit to create one")
	relay := flag.Bool("relay", false, "require TURN relay candidates")
	testAudio := flag.Bool("test-audio", false, "send synthetic Opus silence for RTP diagnostics")
	flag.Parse()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := run(ctx, log, *api, *origin, *room, *relay, *testAudio); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("probe stopped", "error", err)
		os.Exit(1)
	}
}
func run(ctx context.Context, log *slog.Logger, api, origin, room string, forceRelay bool, testAudio bool) error {
	if room == "" {
		req, err := http.NewRequestWithContext(ctx, "POST", api+"/v1/lab/rooms", bytes.NewReader(nil))
		if err != nil {
			return err
		}
		req.Header.Set("Origin", origin)
		client := &http.Client{Timeout: 5 * time.Second}
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != 201 {
			return fmt.Errorf("create room status %d", response.StatusCode)
		}
		var body struct {
			RoomID string `json:"roomId"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			return err
		}
		room = body.RoomID
	}
	log.Info("join this room in the browser networking lab", "room", room)
	wsURL := strings.Replace(strings.Replace(api, "https://", "wss://", 1), "http://", "ws://", 1) + "/v1/lab/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{origin}}})
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	conn.SetReadLimit(64 << 10)
	send := func(kind string, payload any) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body, err := json.Marshal(envelope{Version: 1, Type: kind, MatchID: room, Payload: data})
		if err != nil {
			return err
		}
		deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return conn.Write(deadline, websocket.MessageText, body)
	}
	if err := send("room.join", map[string]string{"roomId": room}); err != nil {
		return err
	}
	var pc *webrtc.PeerConnection
	defer func() {
		if pc != nil {
			_ = pc.Close()
		}
	}()
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeat.C:
				if send("presence.heartbeat", struct{}{}) != nil {
					return
				}
			}
		}
	}()
	pending := []webrtc.ICECandidateInit{}
	bind := func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() { log.Info("DataChannel open"); _ = dc.SendText("Hello from Pion rtc-probe") })
		dc.OnMessage(func(m webrtc.DataChannelMessage) {
			if m.IsString && len(m.Data) <= 2000 {
				log.Info("DataChannel message received", "bytes", len(m.Data))
				if string(m.Data) != "Pion received your message" {
					_ = dc.SendText("Pion received your message")
				}
			}
		})
	}
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		var event envelope
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
		switch event.Type {
		case "connection.ready":
			var config webrtc.Configuration
			if err := json.Unmarshal(event.Payload, &config); err != nil {
				return err
			}
			if forceRelay {
				config.ICETransportPolicy = webrtc.ICETransportPolicyRelay
			}
			pc, err = webrtc.NewPeerConnection(config)
			if err != nil {
				return err
			}
			for _, kind := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeAudio, webrtc.RTPCodecTypeVideo} {
				if _, err := pc.AddTransceiverFromKind(kind, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
					return err
				}
			}
			if testAudio {
				track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2}, "probe-audio", "probe")
				if err != nil {
					return err
				}
				sender, err := pc.AddTrack(track)
				if err != nil {
					return err
				}
				go func() {
					buffer := make([]byte, 1500)
					for {
						if _, _, err := sender.Read(buffer); err != nil {
							return
						}
					}
				}()
				go func() {
					ticker := time.NewTicker(20 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-ctx.Done():
							return
						case <-ticker.C:
							if err := track.WriteSample(media.Sample{Data: []byte{0xf8, 0xff, 0xfe}, Duration: 20 * time.Millisecond}); err != nil {
								return
							}
						}
					}
				}()
			}
			peer := pc
			pc.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c != nil {
					log.Info("ICE candidate", "type", c.Typ.String(), "protocol", c.Protocol.String())
					_ = send("webrtc.ice", c.ToJSON())
				}
			})
			pc.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) { log.Info("ICE gathering", "state", state.String()) })
			pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) { log.Info("ICE connection", "state", state.String()) })
			pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
				log.Info("PeerConnection", "state", state.String())
				if state == webrtc.PeerConnectionStateConnected && peer.SCTP() != nil {
					pair, err := peer.SCTP().Transport().ICETransport().GetSelectedCandidatePair()
					if err == nil && pair != nil {
						log.Info("selected pair", "local_type", pair.Local.Typ.String(), "remote_type", pair.Remote.Typ.String())
					}
				}
			})
			pc.OnDataChannel(bind)
			pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
				log.Info("receiving media", "kind", track.Kind().String(), "codec", track.Codec().MimeType)
				for {
					if _, _, err := track.ReadRTP(); err != nil {
						return
					}
				}
			})
		case "match.found":
			var p struct {
				Offerer bool `json:"offerer"`
			}
			_ = json.Unmarshal(event.Payload, &p)
			if p.Offerer {
				dc, err := pc.CreateDataChannel("chat", nil)
				if err != nil {
					return err
				}
				bind(dc)
				offer, err := pc.CreateOffer(nil)
				if err != nil {
					return err
				}
				if err := pc.SetLocalDescription(offer); err != nil {
					return err
				}
				if err := send("webrtc.offer", offer); err != nil {
					return err
				}
			}
		case "webrtc.offer", "webrtc.answer":
			var description webrtc.SessionDescription
			if err := json.Unmarshal(event.Payload, &description); err != nil {
				return err
			}
			if err := pc.SetRemoteDescription(description); err != nil {
				return err
			}
			for _, candidate := range pending {
				if err := pc.AddICECandidate(candidate); err != nil {
					return err
				}
			}
			pending = nil
			if event.Type == "webrtc.offer" {
				answer, err := pc.CreateAnswer(nil)
				if err != nil {
					return err
				}
				if err := pc.SetLocalDescription(answer); err != nil {
					return err
				}
				if err := send("webrtc.answer", answer); err != nil {
					return err
				}
			}
		case "webrtc.ice":
			var candidate webrtc.ICECandidateInit
			if err := json.Unmarshal(event.Payload, &candidate); err != nil {
				return err
			}
			if pc.RemoteDescription() == nil {
				pending = append(pending, candidate)
			} else if err := pc.AddICECandidate(candidate); err != nil {
				return err
			}
		case "match.ended":
			log.Info("room ended")
			return nil
		case "error":
			return errors.New("signaling rejected the room or event")
		}
	}
}
