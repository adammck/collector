import { useEffect } from 'react';
import type { Output } from '../types';

interface Props {
  output: Output;
  disabled?: boolean;
  onSubmit: (index: number) => void;
}

export function OptionList({ output, disabled = false, onSubmit }: Props) {
  const options = output.OptionList?.options;
  
  useEffect(() => {
    if (!options) return;
    
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
  
  if (!options) return null;
  
  return (
    <div className={`space-y-3 ${disabled ? 'opacity-50' : ''}`}>
      {options.map((option, index) => {
        const hotkey = option.hotkey;
        return (
          <button
            key={index}
            className="w-full px-6 py-4 text-left bg-gradient-to-r from-blue-50 to-indigo-50 hover:from-blue-100 hover:to-indigo-100 border border-blue-200 rounded-xl cursor-pointer disabled:cursor-not-allowed transition-all duration-200 hover:shadow-md transform hover:scale-[1.02] active:scale-[0.98]"
            onClick={() => onSubmit(index)}
            disabled={disabled}
          >
            <div className="flex items-center justify-between">
              <span className="text-lg font-medium text-gray-800">
                {option.label}
              </span>
              {hotkey && (
                <span className="px-2 py-1 bg-white border border-gray-300 rounded text-sm font-mono text-gray-600 shadow-sm">
                  {hotkey}
                </span>
              )}
            </div>
          </button>
        );
      })}
    </div>
  );
}