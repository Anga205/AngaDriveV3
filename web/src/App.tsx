import { Router, Route, useLocation, useNavigate } from "@solidjs/router";
import type { RouteSectionProps } from "@solidjs/router";
import type { Component } from "solid-js";
import HomePage from "./pages/HomePage";
import { MyDrive } from "./pages/MyDrive";
import { useWebSocket, WebSocketProvider } from "./Websockets";
import MyCollections from "./pages/MyCollections";
import Account from "./pages/Account";
import { AppContext, ContextProvider } from "./Context";
import { createEffect, useContext, onMount, onCleanup } from "solid-js";
import { UniversalMessageHandler } from "./library/functions";
import CollectionPage from "./pages/Collection";
import toast from "solid-toast";


// Mounted inside Router so router hooks are valid
let __globalPasteListenerAttached = false;
const GlobalPasteHandler = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const ctx = useContext(AppContext)!;
  onMount(() => {
    if (__globalPasteListenerAttached) return;
    const pasteHandler = (e: ClipboardEvent) => {
      try {
        // If a page-level handler already handled it, skip
        if (e.defaultPrevented) return;
        // Ignore when typing in inputs/textarea/contenteditable
        const ae = document.activeElement as HTMLElement | null;
        if (ae && (ae.tagName === 'INPUT' || ae.tagName === 'TEXTAREA' || ae.isContentEditable)) return;

        const dt = e.clipboardData;
        if (!dt) return;
        const fromFiles = dt.files ? Array.from(dt.files) : [];
        const fromItems = dt.items ? Array.from(dt.items)
          .filter(it => it.kind === 'file')
          .map(it => it.getAsFile())
          .filter((f): f is File => !!f && f.size > 0) : [];
        // Merge and dedupe by name|size|type|lastModified to avoid duplicates
        const seen = new Set<string>();
        const files: File[] = [];
        for (const f of [...fromFiles, ...fromItems]) {
          const key = `${f.name}|${f.size}|${f.type}|${(f as any).lastModified ?? ''}`;
          if (!seen.has(key)) { seen.add(key); files.push(f); }
        }
        if (files.length === 0) {
          // Gracefully ignore pure text pastes globally
          return;
        }

        // Prevent default so page-level handlers don't also process and browser doesn't paste images
        e.preventDefault();

        const path = location.pathname || "";
        if (path.startsWith("/collection")) {
          document.dispatchEvent(new CustomEvent("open-collection-upload", { detail: { files } }));
          return;
        }

        const openDriveUpload = () => {
          document.dispatchEvent(new CustomEvent("open-drive-upload", { detail: { files } }));
        };

        if (path !== "/my_drive") {
          // Use context to pass pending files and route
          ctx.setPendingDriveUploadFiles?.(files);
          try { toast.success(`Opening My Drive to upload ${files.length} file${files.length>1?'s':''}…`); } catch {}
          navigate("/my_drive");
        } else {
          openDriveUpload();
        }
      } catch (err) {
        console.error('Global paste handler error:', err);
      }
    };

    document.addEventListener('paste', pasteHandler);
    __globalPasteListenerAttached = true;
    onCleanup(() => {
      document.removeEventListener('paste', pasteHandler);
      __globalPasteListenerAttached = false;
    });
  });
  return null;
}

const UncontextedApp = () => {
  const { socket: getSocket } = useWebSocket();
  const ctx = useContext(AppContext)!;
  createEffect(() => {
    const socket = getSocket();
    if (!socket) return;
    socket.addEventListener("message", (event) => {
      UniversalMessageHandler(event, ctx);
    });
  })
  return (
    <Router>
      <Route path="/" component={RootLayout}>
        <Route path="/" component={HomePage} />
        <Route path="/my_drive" component={MyDrive}/>
        <Route path="/my_collections" component={MyCollections}/>
        <Route path="/account" component={Account}/>
        <Route path="/collection/*" component={CollectionPage} />
      </Route>
    </Router>
  )
}

const RootLayout: Component<RouteSectionProps<unknown>> = (props) => {
  return (
    <>
      <GlobalPasteHandler />
      {props.children}
    </>
  );
}

const App = () => {
  return (
    <WebSocketProvider>
      <ContextProvider>
        <UncontextedApp />
      </ContextProvider>
    </WebSocketProvider>
  )
}



export default App
