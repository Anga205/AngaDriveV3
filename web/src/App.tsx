import { Router, Route, useLocation, useNavigate } from "@solidjs/router";
import type { RouteSectionProps } from "@solidjs/router";
import type { Component } from "solid-js";
import HomePage from "./pages/HomePage/HomePage";
import { MyDrive } from "./pages/MyDrive/MyDrive";
import { useWebSocket, WebSocketProvider } from "./Websockets";
import MyCollections from "./pages/MyCollections/MyCollections";
import Account from "./pages/Account/Account";
import { AppContext, ContextProvider } from "./Context";
import { createEffect, useContext, onMount, onCleanup, Show } from "solid-js";
import { UniversalMessageHandler } from "./library/functions";
import CollectionPage from "./pages/Collection/Collection";
import MobileTabShell from "./components/MobileTabShell";
import { useIsMobile } from "./library/useIsMobile";
import { MOBILE_TABS, indexFromPath } from "./library/tabs";
import { Toaster } from "solid-toast";
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
  const location = useLocation();
  const isMobile = useIsMobile();

  // On mobile the four tab pages live inside a persistent swipeable pager
  // (`MobileTabShell`) instead of the router outlet, so that they can sit side
  // by side and keep their scroll positions. The outlet is only used for
  // desktop, and for mobile pages that are not tabs (collection detail pages),
  // which are rendered as a full-screen overlay above the pager.
  const isCollectionRoute = () => location.pathname.startsWith("/collection");

  // All four tab pages are mounted at once on mobile, so their own <title>
  // elements cannot decide the document title. Own it here instead.
  createEffect(() => {
    document.title = isCollectionRoute()
      ? "Collection | DriveV3"
      : MOBILE_TABS[indexFromPath(location.pathname)].title;
  });

  return (
    <>
      <GlobalPasteHandler />
      <Show when={isMobile()}>
        <MobileTabShell />
      </Show>
      {/* Desktop: the router outlet renders the page normally. */}
      <Show when={!isMobile()}>{props.children}</Show>
      {/* Mobile non-tab pages (collection detail) overlay the pager, but stay
          below the fixed navbar so the user can still switch tabs. */}
      <Show when={isMobile() && isCollectionRoute()}>
        <div class="fixed inset-0 z-20 bg-black">{props.children}</div>
      </Show>
      <Toaster
        position="bottom-right"
        gutter={8}
        containerClassName=""
        containerStyle={{}}
        toastOptions={{
          className: '',
          duration: 2000,
          style: {
            background: '#363636',
            color: '#fff',
          },
        }}
      />
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
