import { useEffect } from 'react';

interface Props {
  onDefer: () => void;
  onNext: () => void;
}

export function useKeyboardShortcuts({ onDefer, onNext }: Props) {
  useEffect(() => {
    const handleKeydown = (e: KeyboardEvent) => {
      if (e.key === 'd' && e.ctrlKey) {
        e.preventDefault();
        onDefer();
      }
      if (e.key === 'n' && e.ctrlKey) {
        e.preventDefault();
        onNext();
      }
    };
    
    document.addEventListener('keydown', handleKeydown);
    return () => document.removeEventListener('keydown', handleKeydown);
  }, [onDefer, onNext]);
}