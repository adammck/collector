import { create } from 'zustand';
import type { AppState, Queue } from './types';

interface AppStore {
  currentUUID: string | null;
  state: AppState;
  stateMessage: string | null;
  queue: Queue | null;
  
  setCurrentUUID: (uuid: string | null) => void;
  setState: (state: AppState, message?: string) => void;
  setQueue: (queue: Queue | null) => void;
}

export const useAppStore = create<AppStore>((set) => ({
  currentUUID: null,
  state: 'idle',
  stateMessage: null,
  queue: null,
  
  setCurrentUUID: (uuid) => set({ currentUUID: uuid }),
  setState: (state, message) => set({ state, stateMessage: message || null }),
  setQueue: (queue) => set({ queue }),
}));