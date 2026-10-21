import { createSignal } from "solid-js";

/**
 * Single shared mobile breakpoint for the whole app.
 *
 * Previously each page kept its own `window.innerWidth` signal and listener
 * (and HomePage used 640 while everything else used 768). The mobile tab
 * pager needs one consistent answer, otherwise at widths between 640 and 768
 * the shell would render mobile panes that contain desktop page layouts.
 */
const MOBILE_BREAKPOINT = 768;

const query =
    typeof window !== "undefined"
        ? window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT}px)`)
        : undefined;

const [isMobile, setIsMobile] = createSignal(query ? query.matches : false);

if (query) {
    query.addEventListener("change", (event) => setIsMobile(event.matches));
}

/** Reactive accessor: `true` when the viewport is at or below the mobile breakpoint. */
const useIsMobile = () => isMobile;

export { useIsMobile, MOBILE_BREAKPOINT };
