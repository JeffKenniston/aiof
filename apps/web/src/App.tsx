import React from 'react';
import { AppShell, Chat, ContextMenu } from '@aiof/ui';

class ErrorBoundary extends React.Component<{children: any}, {hasError: boolean, error: any}> {
  constructor(props: any) { super(props); this.state = { hasError: false, error: null }; }
  static getDerivedStateFromError(error: any) { return { hasError: true, error }; }
  render() {
    if (this.state.hasError) {
      return <div style={{color: 'red', padding: 20}}><pre>{String(this.state.error?.stack || this.state.error)}</pre></div>;
    }
    return this.props.children;
  }
}

export default function App() {
  return (
    <ErrorBoundary>
      <div className="w-screen h-screen flex flex-col bg-[#090d16] text-slate-100 overflow-hidden">
        <AppShell>
           <Chat hideHeader={false} transparentBg={true} />
        </AppShell>
        <ContextMenu />
      </div>
    </ErrorBoundary>
  );
}
