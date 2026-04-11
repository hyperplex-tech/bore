/// <reference types="vite/client" />

interface Window {
  runtime?: {
    EventsOn: (eventName: string, callback: (...data: any[]) => void) => () => void;
    EventsOff: (eventName: string) => void;
    EventsEmit: (eventName: string, ...data: any[]) => void;
    Quit: () => void;
  };
  go?: Record<string, Record<string, Record<string, (...args: any[]) => Promise<any>>>>;
}
