import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { AppShell, Chat, ContextMenu } from '@aiof/ui';
export default function App() {
    return (_jsxs("div", { className: "w-screen h-screen flex flex-col bg-[#090d16] text-slate-100 overflow-hidden", children: [_jsx(AppShell, { children: _jsx(Chat, { hideHeader: false, transparentBg: true }) }), _jsx(ContextMenu, {})] }));
}
