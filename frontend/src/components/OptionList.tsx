import { useEffect } from 'react';
import type { Output } from '../types';

interface Props {
  output: Output;
  disabled?: boolean;
  onSubmit: (index: number) => void;
}

export function OptionList({ output, disabled = false, onSubmit }: Props) {
  const options = output.OptionList?.options;
  
  if (!options) return null;
  
  useEffect(() => {
    const hotkeyMap = new Map<string, number>();
    options.forEach((opt, i) => {
      if (opt.hotkey) hotkeyMap.set(opt.hotkey, i);
    });
    
    const handleKeydown = (e: KeyboardEvent) => {
      if (disabled) return;
      
      const index = hotkeyMap.get(e.key);
      if (index !== undefined) {
        e.preventDefault();
        onSubmit(index);
      }
    };
    
    document.addEventListener('keydown', handleKeydown);
    return () => document.removeEventListener('keydown', handleKeydown);
  }, [options, disabled, onSubmit]);
  
  return (
    <div className={`space-y-1 ${disabled ? 'opacity-20' : ''}`}>
      {options.map((option, index) => (
        <div key={index}>
          <button
            className="px-4 py-2 text-lg bg-gray-100 hover:bg-gray-200 border rounded cursor-pointer disabled:cursor-not-allowed"
            onClick={() => onSubmit(index)}
            disabled={disabled}
          >
            {option.label}
          </button>
        </div>
      ))}
    </div>
  );
}