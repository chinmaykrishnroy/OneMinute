// Send the same selfie-oriented frames that the local preview displays.
// The canvas follows camera dimensions, including portrait/landscape changes.
import { processCamera } from "./processed-camera";
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
  return processCamera(raw, () => facing === "user");
}
