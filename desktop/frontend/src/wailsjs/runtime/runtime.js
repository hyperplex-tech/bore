// Wails runtime is injected at build time. This stub provides the API shape
// for development. At runtime, window.runtime is the real implementation.
export function EventsOn(eventName, callback) {
  return window.runtime?.EventsOn(eventName, callback) ?? (() => {});
}
export function EventsOff(eventName) {
  window.runtime?.EventsOff(eventName);
}
export function EventsEmit(eventName, ...data) {
  window.runtime?.EventsEmit(eventName, ...data);
}
export function Quit() {
  window.runtime?.Quit();
}
