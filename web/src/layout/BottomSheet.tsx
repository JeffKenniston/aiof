import React from 'react';

interface BottomSheetProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}

export default function BottomSheet({ isOpen, onClose, title, children }: BottomSheetProps) {
  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop overlay */}
      <div 
        className="fixed inset-0 bg-black/50 backdrop-blur-sm z-overlay transition-opacity"
        onClick={onClose}
      />
      
      {/* Sheet */}
      <div 
        className={`fixed bottom-0 left-0 right-0 bg-surface-raised rounded-t-2xl z-modal transform transition-transform duration-300 ease-out max-h-[85vh] flex flex-col shadow-elevated border-t border-border-default`}
        style={{ transform: isOpen ? 'translateY(0)' : 'translateY(100%)' }}
      >
        <div className="flex-none p-3 flex justify-center cursor-pointer" onClick={onClose}>
          <div className="w-12 h-1.5 bg-border-default rounded-full" />
        </div>
        
        {title && (
          <div className="px-4 pb-2 border-b border-border-subtle flex justify-between items-center">
            <h3 className="font-semibold text-text-primary text-lg">{title}</h3>
            <button onClick={onClose} className="text-text-muted hover:text-text-primary p-1">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
            </button>
          </div>
        )}
        
        <div className="flex-1 overflow-y-auto p-4">
          {children}
        </div>
      </div>
    </>
  );
}
