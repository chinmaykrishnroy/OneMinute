export type MediaSource = "camera" | "test";
export type Capture = { stream: MediaStream; stop: () => void };

export async function acquireMedia(source: MediaSource): Promise<Capture> {
  if (source === "camera") {
    const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
    return { stream, stop: () => stream.getTracks().forEach(track => track.stop()) };
  }
  // A local diagnostic source lets tests exercise real RTP without recording a person.
  const canvas = document.createElement("canvas");
  canvas.width = 640; canvas.height = 360;
  const context = canvas.getContext("2d");
  if (!context) throw new Error("Canvas is unavailable.");
  const hue = Math.floor(Math.random() * 360);
  const draw = () => {
    context.fillStyle = `hsl(${hue} 45% 22%)`; context.fillRect(0, 0, 640, 360);
    context.fillStyle = "#fff"; context.font = "28px monospace";
    context.fillText("WebRTC test pattern", 35, 150);
    context.fillText(new Date().toLocaleTimeString(), 35, 205);
  };
  draw();
  const interval = window.setInterval(draw, 100);
  const stream = canvas.captureStream(10);
  let audio: AudioContext | undefined;
  try {
    audio = new AudioContext();
    const oscillator = audio.createOscillator();
    const gain = audio.createGain(); gain.gain.value = 0;
    const destination = audio.createMediaStreamDestination();
    oscillator.connect(gain).connect(destination); oscillator.start();
    stream.addTrack(destination.stream.getAudioTracks()[0]);
    const audioContext = audio;
    return { stream, stop: () => { clearInterval(interval); stream.getTracks().forEach(track => track.stop()); oscillator.stop(); void audioContext.close(); } };
  } catch (error) {
    clearInterval(interval); stream.getTracks().forEach(track => track.stop()); void audio?.close(); throw error;
  }
}
