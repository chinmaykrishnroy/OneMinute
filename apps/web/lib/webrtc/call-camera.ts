// Send the same selfie-oriented frames that the local preview displays.
// The canvas follows camera dimensions, including portrait/landscape changes.
export async function callCamera(
  facing: string,
): Promise<{ track: MediaStreamTrack; stop: () => void }> {
  const raw = await navigator.mediaDevices.getUserMedia({
    video: {
      facingMode: { ideal: facing },
      width: { ideal: 1920 },
      height: { ideal: 1080 },
      frameRate: { ideal: 30, max: 30 },
    },
  });
  const source = document.createElement("video");
  source.muted = true;
  source.playsInline = true;
  source.srcObject = raw;
  const canvas = document.createElement("canvas");
  const context = canvas.getContext("2d");
  try {
    if (!context) throw new Error("Camera processing unavailable");
    await source.play();
  } catch (error) {
    raw.getTracks().forEach((t) => t.stop());
    source.srcObject = null;
    throw error;
  }
  const draw = () => {
    if (!source.videoWidth || !source.videoHeight || !context) return;
    if (
      canvas.width !== source.videoWidth ||
      canvas.height !== source.videoHeight
    ) {
      canvas.width = source.videoWidth;
      canvas.height = source.videoHeight;
    }
    context.save();
    if (facing === "user") {
      context.translate(canvas.width, 0);
      context.scale(-1, 1);
    }
    context.drawImage(source, 0, 0, canvas.width, canvas.height);
    context.restore();
  };
  draw();
  const output = canvas.captureStream(30);
  const track = output.getVideoTracks()[0];
  track.contentHint = "motion";
  const timer = window.setInterval(draw, 1000 / 30);
  return {
    track,
    stop: () => {
      clearInterval(timer);
      track.stop();
      raw.getTracks().forEach((t) => t.stop());
      source.pause();
      source.srcObject = null;
    },
  };
}
