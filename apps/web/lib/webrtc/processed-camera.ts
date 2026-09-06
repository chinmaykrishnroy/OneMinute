type Processor = { readable: ReadableStream<VideoFrame> };
type Generator = MediaStreamTrack & { writable: WritableStream<VideoFrame> };
type FrameAPIs = {
  MediaStreamTrackProcessor?: new (options: {
    track: MediaStreamTrack;
    maxBufferSize: number;
  }) => Processor;
  MediaStreamTrackGenerator?: new (options: { kind: "video" }) => Generator;
};
export type CameraCapture = { track: MediaStreamTrack; stop: () => void };

// A single pending frame and the original capture timestamp keep processed
// video current and on the microphone's clock. Stop owns video, never audio.
export async function processCamera(
  raw: MediaStream,
  mirror: () => boolean,
): Promise<CameraCapture> {
  const input = raw.getVideoTracks()[0];
  const apis = window as unknown as FrameAPIs;
  if (apis.MediaStreamTrackProcessor && apis.MediaStreamTrackGenerator) {
    let reader: ReadableStreamDefaultReader<VideoFrame> | undefined;
    let writer: WritableStreamDefaultWriter<VideoFrame> | undefined;
    let output: Generator | undefined;
    try {
      const processor = new apis.MediaStreamTrackProcessor({
        track: input,
        maxBufferSize: 1,
      });
      output = new apis.MediaStreamTrackGenerator({ kind: "video" });
      const canvas = new OffscreenCanvas(1, 1);
      const context = canvas.getContext("2d", {
        alpha: false,
        desynchronized: true,
      });
      if (!context) throw new Error("Frame processing unavailable");
      reader = processor.readable.getReader();
      writer = output.writable.getWriter();
      const frameReader = reader,
        frameWriter = writer,
        track = output;
      let stopped = false;
      const stop = () => {
        if (stopped) return;
        stopped = true;
        void frameReader.cancel().catch(() => {});
        void frameWriter.abort().catch(() => {});
        track.stop();
        input.stop();
      };
      void (async () => {
        try {
          while (!stopped) {
            await frameWriter.ready;
            const { value: frame, done } = await frameReader.read();
            if (done || !frame) break;
            try {
              if (stopped) break;
              if (
                canvas.width !== frame.displayWidth ||
                canvas.height !== frame.displayHeight
              ) {
                canvas.width = frame.displayWidth;
                canvas.height = frame.displayHeight;
              }
              context.save();
              if (mirror()) {
                context.translate(canvas.width, 0);
                context.scale(-1, 1);
              }
              context.drawImage(frame, 0, 0, canvas.width, canvas.height);
              context.restore();
              const transformed = new VideoFrame(canvas, {
                timestamp: frame.timestamp,
              });
              try {
                await frameWriter.write(transformed);
              } finally {
                transformed.close();
              }
            } finally {
              frame.close();
            }
          }
        } catch {
          /* Cancellation/device removal ends the bounded stream. */
        } finally {
          stop();
        }
      })();
      track.contentHint = "motion";
      return { track, stop };
    } catch {
      void reader?.cancel().catch(() => {});
      void writer?.abort().catch(() => {});
      output?.stop();
    }
  }
  // Compatibility path: request one captured output for each fresh camera
  // frame. No fixed timer duplicates old frames or builds a redraw backlog.
  const source = document.createElement("video");
  source.muted = true;
  source.playsInline = true;
  source.srcObject = new MediaStream([input]);
  const canvas = document.createElement("canvas");
  const context = canvas.getContext("2d", {
    alpha: false,
    desynchronized: true,
  });
  try {
    if (!context) throw new Error("Camera processing unavailable");
    await source.play();
  } catch (error) {
    input.stop();
    source.srcObject = null;
    throw error;
  }
  let stopped = false,
    callback = 0,
    animation = 0,
    lastTime = -1;
  const supportsFrames = typeof source.requestVideoFrameCallback === "function";
  const output = canvas.captureStream(0);
  const track = output.getVideoTracks()[0] as CanvasCaptureMediaStreamTrack;
  const automatic =
    typeof track.requestFrame !== "function" ? canvas.captureStream(30) : null;
  const result = automatic?.getVideoTracks()[0] ?? track;
  if (automatic) track.stop();
  const draw = () => {
    if (stopped) return;
    if (
      source.videoWidth &&
      source.videoHeight &&
      source.currentTime !== lastTime
    ) {
      lastTime = source.currentTime;
      if (
        canvas.width !== source.videoWidth ||
        canvas.height !== source.videoHeight
      ) {
        canvas.width = source.videoWidth;
        canvas.height = source.videoHeight;
      }
      context!.save();
      if (mirror()) {
        context!.translate(canvas.width, 0);
        context!.scale(-1, 1);
      }
      context!.drawImage(source, 0, 0, canvas.width, canvas.height);
      context!.restore();
      if (!automatic) track.requestFrame();
    }
    if (supportsFrames) callback = source.requestVideoFrameCallback(draw);
    else animation = requestAnimationFrame(draw);
  };
  draw();
  result.contentHint = "motion";
  return {
    track: result,
    stop: () => {
      stopped = true;
      if (supportsFrames) source.cancelVideoFrameCallback(callback);
      cancelAnimationFrame(animation);
      result.stop();
      input.stop();
      source.pause();
      source.srcObject = null;
    },
  };
}
