// The answerer binds local tracks to the transceivers in the remote offer.
// An extra transceiver created beforehand can be absent from the answer.
export async function attachCallMedia(
  pc: RTCPeerConnection,
  media: MediaStream,
  answering: boolean,
) {
  let videoSender: RTCRtpSender | undefined;
  for (const kind of ["audio", "video"] as const) {
    const track = media.getTracks().find((item) => item.kind === kind) ?? null;
    const transceiver = answering
      ? pc.getTransceivers().find((item) => item.receiver.track.kind === kind)
      : pc.addTransceiver(track ?? kind, {
          direction: "sendrecv",
          streams: [media],
        });
    if (!transceiver) throw new Error(`The call did not offer ${kind}`);
    if (answering) {
      transceiver.direction = "sendrecv";
      transceiver.sender.setStreams(media);
      await transceiver.sender.replaceTrack(track);
    }
    if (kind === "video") {
      videoSender = transceiver.sender;
      const parameters = videoSender.getParameters();
      parameters.degradationPreference = "balanced";
      await videoSender.setParameters(parameters);
    }
  }
  return videoSender!;
}

export function receiveTrack(
  stream: MediaStream,
  event: RTCTrackEvent,
  element: HTMLVideoElement | null,
) {
  if (!stream.getTracks().some((track) => track.id === event.track.id))
    stream.addTrack(event.track);
  if (element && element.srcObject !== stream) element.srcObject = stream;
}
