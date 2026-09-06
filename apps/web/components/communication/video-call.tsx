"use client";
import { useEffect, useRef, useState, PointerEvent } from "react";
import { useCommunication, LiveEvent } from "./runtime";
import { callCamera } from "@/lib/webrtc/call-camera";
export type CallSession = {
  id: string;
  connectionId: string;
  caller: string;
  callee: string;
  name: string;
  video: boolean;
  outgoing: boolean;
};

export function CallOverlay({
  call,
  ice,
  onClose,
}: {
  call: CallSession;
  ice: RTCConfiguration;
  onClose: () => void;
}) {
  const { api, subscribe } = useCommunication();
  const [phase, setPhase] = useState("ringing");
  const [status, setStatus] = useState(
    call.outgoing ? "Calling..." : "Incoming call",
  );
  const [videoOn, setVideoOn] = useState(call.video);
  const [muted, setMuted] = useState(false);
  const [remoteVideoOn, setRemoteVideoOn] = useState(call.video);
  const [position, setPosition] = useState({ x: 0.72, y: 0.1 });
  const [facing, setFacing] = useState("user");
  const [localRatio, setLocalRatio] = useState(3 / 4);
  const surface = useRef<HTMLDialogElement>(null);
  const localElement = useRef<HTMLVideoElement>(null);
  const remoteElement = useRef<HTMLVideoElement>(null);
  const peer = useRef<RTCPeerConnection | null>(null);
  const stream = useRef<MediaStream | null>(null);
  const sender = useRef<RTCRtpSender | null>(null);
  const pending = useRef<RTCIceCandidateInit[]>([]);
  const dead = useRef(false);
  const started = useRef(false);
  const callPhase = useRef("ringing");
  const closeRef = useRef(onClose);
  useEffect(() => {
    closeRef.current = onClose;
  }, [onClose]);
  const handling = useRef<Promise<void>>(Promise.resolve());
  const camera = useRef<Awaited<ReturnType<typeof callCamera>> | null>(null);
  async function action(type: string, payload: unknown = {}) {
    const response = await fetch(new URL(`/v1/calls/${call.id}`, api), {
      method: "POST",
      credentials: "include",
      keepalive: type === "end" || type === "decline",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type, payload }),
    });
    if (!response.ok) throw new Error("Call unavailable");
  }
  function cleanup() {
    dead.current = true;
    camera.current?.stop();
    camera.current = null;
    stream.current?.getTracks().forEach((track) => track.stop());
    peer.current?.close();
    peer.current = null;
  }
  async function finish(type = "end") {
    try {
      await action(type);
    } catch {}
    cleanup();
    closeRef.current();
  }
  async function setup() {
    if (started.current) return;
    started.current = true;
    try {
      const media = await navigator.mediaDevices.getUserMedia({ audio: true });
      if (dead.current) {
        media.getTracks().forEach((t) => t.stop());
        return;
      }
      stream.current = media;
      if (call.video) {
        const next = await callCamera("user");
        if (dead.current) {
          next.stop();
          return;
        }
        camera.current = next;
        media.addTrack(next.track);
      }
      stream.current = media;
      if (localElement.current) localElement.current.srcObject = media;
      const connection = new RTCPeerConnection(ice);
      peer.current = connection;
      media
        .getAudioTracks()
        .forEach((track) => connection.addTrack(track, media));
      sender.current = connection.addTransceiver(
        media.getVideoTracks()[0] || "video",
        { direction: "sendrecv", streams: [media] },
      ).sender;
      connection.ontrack = (e) => {
        if (remoteElement.current)
          remoteElement.current.srcObject =
            e.streams[0] || new MediaStream([e.track]);
      };
      connection.onicecandidate = (e) => {
        if (e.candidate)
          void action("ice", e.candidate.toJSON()).catch(() => {});
      };
      connection.onconnectionstatechange = () => {
        if (connection.connectionState === "connected") {
          setPhase("active");
          setStatus("Connected");
        } else if (connection.connectionState === "failed") {
          setStatus("The connection was lost.");
          void finish();
        }
      };
      callPhase.current = "connecting";
      setPhase("connecting");
      setStatus("Connecting...");
    } catch {
      setStatus(
        "Camera or microphone access failed. Check your permissions and try again.",
      );
      try {
        await action("end");
      } catch {}
      cleanup();
      setPhase("ended");
    }
  }
  async function accept() {
    try {
      await setup();
      if (!dead.current) await action("accept");
    } catch {
      cleanup();
      setStatus("This call has ended.");
      setPhase("ended");
    }
  }
  useEffect(() => {
    surface.current?.showModal();
    const videoElement = localElement.current;
    const resizePreview = () => {
      if (videoElement?.videoWidth && videoElement.videoHeight)
        setLocalRatio(videoElement.videoWidth / videoElement.videoHeight);
    };
    videoElement?.addEventListener("resize", resizePreview);
    const pageHide = () => {
      if (!dead.current) void finish();
    };
    window.addEventListener("pagehide", pageHide);
    const unsubscribe = subscribe((event: LiveEvent) => {
      if (event.type === "transport.closed") {
        cleanup();
        setPhase("ended");
        setStatus("Connection lost. Please call again when reconnected.");
        return;
      }
      if (event.call?.id !== call.id) return;
      handling.current = handling.current
        .then(async () => {
          if (dead.current) return;
          if (
            event.type === "call.accept" &&
            !call.outgoing &&
            callPhase.current === "ringing"
          ) {
            cleanup();
            setPhase("ended");
            setStatus("Answered on another tab or device.");
            return;
          }
          if (event.type === "call.end" || event.type === "call.decline") {
            cleanup();
            setPhase("ended");
            setStatus(
              event.type === "call.decline" ? "Call declined" : "Call ended",
            );
            return;
          }
          if (event.type === "call.accept" && call.outgoing) {
            await setup();
            const pc = peer.current;
            if (!pc) return;
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);
            await action("offer", offer);
          }
          if (event.type === "call.offer") {
            await setup();
            const pc = peer.current;
            if (!pc) return;
            await pc.setRemoteDescription(
              event.payload as RTCSessionDescriptionInit,
            );
            for (const candidate of pending.current.splice(0))
              await pc.addIceCandidate(candidate);
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            await action("answer", answer);
          }
          if (event.type === "call.answer" && peer.current) {
            await peer.current.setRemoteDescription(
              event.payload as RTCSessionDescriptionInit,
            );
            for (const candidate of pending.current.splice(0))
              await peer.current.addIceCandidate(candidate);
          }
          if (event.type === "call.ice") {
            if (peer.current?.remoteDescription)
              await peer.current.addIceCandidate(
                event.payload as RTCIceCandidateInit,
              );
            else pending.current.push(event.payload as RTCIceCandidateInit);
          }
          if (event.type === "call.media")
            setRemoteVideoOn(
              Boolean((event.payload as { video: boolean }).video),
            );
        })
        .catch(() => {
          setStatus("Could not connect this call. Please try again.");
          void finish();
        });
    });
    const timeout = setTimeout(() => {
      if (callPhase.current === "ringing" && !dead.current) void finish();
    }, 44000);
    const connectingTimeout = setTimeout(() => {
      if (
        peer.current?.connectionState !== "connected" &&
        callPhase.current !== "ringing" &&
        !dead.current
      ) {
        void action("end").catch(() => {});
        cleanup();
        setPhase("ended");
        setStatus("The call could not connect. Please try again.");
      }
    }, 90000);
    const heartbeat = setInterval(() => {
      if (callPhase.current !== "ringing" && !dead.current)
        void action("heartbeat").catch(() => {
          cleanup();
          setPhase("ended");
          setStatus("Call ended. Please call again.");
        });
    }, 20000);
    return () => {
      unsubscribe();
      videoElement?.removeEventListener("resize", resizePreview);
      window.removeEventListener("pagehide", pageHide);
      clearTimeout(timeout);
      clearTimeout(connectingTimeout);
      clearInterval(heartbeat);
      if (!dead.current) void action("end").catch(() => {});
      cleanup();
    };
    // The call session is immutable; mutable media and callbacks live in refs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [call.id]);
  async function toggleVideo() {
    if (!stream.current || !sender.current) return;
    try {
      if (videoOn) {
        await sender.current.replaceTrack(null);
        camera.current?.stop();
        camera.current = null;
        stream.current.getVideoTracks().forEach((t) => {
          t.stop();
          stream.current?.removeTrack(t);
        });
      } else {
        const next = await callCamera(facing);
        if (dead.current) {
          next.stop();
          return;
        }
        try {
          await sender.current.replaceTrack(next.track);
        } catch (error) {
          next.stop();
          throw error;
        }
        camera.current = next;
        stream.current.addTrack(next.track);
        if (localElement.current)
          localElement.current.srcObject = new MediaStream(
            stream.current.getTracks(),
          );
      }
      setVideoOn(!videoOn);
      await action("media", { video: !videoOn, audio: !muted });
    } catch {
      setStatus("Could not change the camera. Check device permissions.");
    }
  }
  async function switchCamera() {
    if (!videoOn || !sender.current || !stream.current) return;
    const next = facing === "user" ? "environment" : "user";
    try {
      const capture = await callCamera(next);
      if (dead.current) {
        capture.stop();
        return;
      }
      try {
        await sender.current.replaceTrack(capture.track);
      } catch (error) {
        capture.stop();
        throw error;
      }
      camera.current?.stop();
      camera.current = capture;
      stream.current.getVideoTracks().forEach((t) => {
        t.stop();
        stream.current?.removeTrack(t);
      });
      stream.current.addTrack(capture.track);
      setFacing(next);
      if (localElement.current)
        localElement.current.srcObject = new MediaStream(
          stream.current.getTracks(),
        );
    } catch {
      setStatus("Could not switch cameras.");
    }
  }
  function move(event: PointerEvent<HTMLDivElement>) {
    if (
      !event.currentTarget.hasPointerCapture(event.pointerId) ||
      !surface.current
    )
      return;
    const bounds = surface.current.getBoundingClientRect();
    const dock = event.currentTarget.getBoundingClientRect();
    setPosition({
      x: Math.max(
        0,
        Math.min(
          (bounds.width - dock.width) / bounds.width,
          (event.clientX - bounds.left - dock.width / 2) / bounds.width,
        ),
      ),
      y: Math.max(
        0,
        Math.min(
          (bounds.height - dock.height - 110) / bounds.height,
          (event.clientY - bounds.top - dock.height / 2) / bounds.height,
        ),
      ),
    });
  }
  return (
    <dialog
      onCancel={(e) => {
        e.preventDefault();
        void finish();
      }}
      className="private-call"
      role="dialog"
      aria-modal="true"
      aria-label={`Call with ${call.name}`}
      ref={surface}
    >
      <video
        className="call-remote-video"
        ref={remoteElement}
        autoPlay
        playsInline
        style={{ opacity: remoteVideoOn ? 1 : 0 }}
      />
      <div className="call-identity">
        <span className="call-avatar">{call.name[0]}</span>
        <h2>{call.name}</h2>
        <p role="status">{status}</p>
        {!remoteVideoOn && <p>Camera is off</p>}
      </div>
      <div
        className="call-local-dock"
        role="group"
        aria-label="Your preview. Drag to move, or use arrow keys."
        tabIndex={0}
        style={{
          left: `clamp(8px, ${position.x * 100}%, calc(100% - clamp(120px, 20vw, 220px) - 8px))`,
          top: `clamp(8px, ${position.y * 100}%, calc(100% - 32dvh - 110px))`,
          aspectRatio: localRatio,
          height: "auto",
          width: "clamp(120px, 20vw, 220px)",
        }}
        onPointerDown={(e) => e.currentTarget.setPointerCapture(e.pointerId)}
        onPointerMove={move}
        onKeyDown={(e) => {
          const d: Record<string, number[]> = {
            ArrowLeft: [-0.05, 0],
            ArrowRight: [0.05, 0],
            ArrowUp: [0, -0.05],
            ArrowDown: [0, 0.05],
          };
          if (d[e.key]) {
            e.preventDefault();
            const [x, y] = d[e.key];
            setPosition((p) => ({
              x: Math.max(0, Math.min(0.65, p.x + x)),
              y: Math.max(0, Math.min(0.6, p.y + y)),
            }));
          }
        }}
      >
        <video
          ref={localElement}
          autoPlay
          muted
          playsInline
          style={{ opacity: videoOn ? 1 : 0 }}
        />
        <span>You{muted ? " (muted)" : ""}</span>
      </div>
      <div className="call-controls">
        {phase === "ringing" && !call.outgoing && (
          <button onClick={() => void accept()}>
            Accept {call.video ? "video" : "audio"} call
          </button>
        )}
        {phase !== "ringing" && phase !== "ended" && (
          <>
            <button
              aria-pressed={muted}
              onClick={() => {
                stream.current?.getAudioTracks().forEach((t) => {
                  t.enabled = muted;
                });
                setMuted(!muted);
                void action("media", { video: videoOn, audio: muted }).catch(
                  () => setStatus("Could not update microphone status."),
                );
              }}
            >
              {muted ? "Unmute" : "Mute"}
            </button>
            <button aria-pressed={videoOn} onClick={() => void toggleVideo()}>
              {videoOn ? "Camera off" : "Camera on"}
            </button>
            {videoOn && (
              <button onClick={() => void switchCamera()}>Flip camera</button>
            )}
          </>
        )}
        <button
          className="danger-button"
          onClick={() =>
            phase === "ended"
              ? closeRef.current()
              : void finish(
                  call.outgoing || phase !== "ringing" ? "end" : "decline",
                )
          }
        >
          {phase === "ended"
            ? "Close"
            : phase === "ringing" && !call.outgoing
              ? "Decline"
              : "End call"}
        </button>
      </div>
    </dialog>
  );
}
