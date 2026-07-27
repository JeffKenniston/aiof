import { useState, useEffect, useRef } from 'react';

export interface ContextMenuItem {
  label: string;
  icon?: React.ReactNode;
  action?: () => void;
  divider?: boolean;
  danger?: boolean;
}

export default function ContextMenu() {
  const [visible, setVisible] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const [items, setItems] = useState<ContextMenuItem[]>([]);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleContextMenu = (e: any) => {
      setPos({ x: e.detail.x, y: e.detail.y });
      setItems(e.detail.items || []);
      setVisible(true);
    };

    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setVisible(false);
      }
    };

    const handleScroll = () => {
      setVisible(false);
    };

    window.addEventListener('open-context-menu', handleContextMenu);
    document.addEventListener('click', handleClick);
    document.addEventListener('scroll', handleScroll, true);

    return () => {
      window.removeEventListener('open-context-menu', handleContextMenu);
      document.removeEventListener('click', handleClick);
      document.removeEventListener('scroll', handleScroll, true);
    };
  }, []);

  if (!visible) return null;

  return (
    <div
      ref={menuRef}
      className="fixed z-50 bg-surface-raised border border-border-default rounded-lg shadow-xl shadow-black/50 overflow-hidden min-w-[160px] py-1 text-sm animate-in fade-in zoom-in-95 duration-100"
      style={{
        left: pos.x,
        top: pos.y,
        // Ensure menu doesn't overflow right/bottom screen edges
        transform: `translate(${pos.x + 160 > window.innerWidth ? '-100%' : '0'}, ${pos.y + (items.length * 32) > window.innerHeight ? '-100%' : '0'})`
      }}
    >
      {items.map((item, idx) => {
        if (item.divider) {
          return <div key={idx} className="h-px bg-border-default my-1" />;
        }
        
        return (
          <button
            key={idx}
            className={`w-full text-left px-3 py-1.5 flex items-center gap-2 transition-colors ${
              item.danger 
                ? 'text-accent-red hover:bg-accent-red hover:text-white' 
                : 'text-text-primary hover:bg-accent-blue hover:text-white'
            }`}
            onClick={() => {
              if (item.action) item.action();
              setVisible(false);
            }}
          >
            {item.icon && <span className="w-4 h-4 flex items-center justify-center">{item.icon}</span>}
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
