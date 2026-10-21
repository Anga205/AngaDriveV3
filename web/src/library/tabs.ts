import { createSignal } from "solid-js";
import type { Component } from "solid-js";
import { CollectionSVG, FileSVG, HomeSVG, UserSVG } from "../assets/SvgFiles";
import type { Pages } from "./types";

/**
 * The mobile bottom-navbar tabs, in pager order (left to right).
 * This array is the single source of truth for both the navbar buttons and
 * the order of the panes inside `MobileTabShell`.
 */
interface MobileTab {
    page: Pages;
    label: string;
    route: string;
    title: string;
    icon: Component;
}

const MOBILE_TABS: ReadonlyArray<MobileTab> = [
    { page: "Home", label: "Home", route: "/", title: "HomePage | DriveV3", icon: HomeSVG },
    { page: "Files", label: "Files", route: "/my_drive", title: "My Files | DriveV3", icon: FileSVG },
    { page: "Collections", label: "Collections", route: "/my_collections", title: "My Collections | DriveV3", icon: CollectionSVG },
    { page: "Account", label: "Account", route: "/account", title: "Account | DriveV3", icon: UserSVG },
];

/**
 * Maps a pathname to the navbar tab that should be highlighted.
 * Collection detail pages keep the Collections tab lit.
 */
const pageFromPath = (pathname: string): Pages => {
    if (pathname.startsWith("/collection")) return "Collections";
    if (pathname.startsWith("/my_drive")) return "Files";
    if (pathname.startsWith("/my_collections")) return "Collections";
    if (pathname.startsWith("/account")) return "Account";
    return "Home";
};

/**
 * Maps a pathname to its pane index inside the mobile pager.
 * Unknown paths fall back to the first pane (Home).
 */
const indexFromPath = (pathname: string): number => {
    const page = pageFromPath(pathname);
    const index = MOBILE_TABS.findIndex((tab) => tab.page === page);
    return index === -1 ? 0 : index;
};

/**
 * Live, fractional position of the mobile tab pager.
 *
 * `0` = first pane (Home), `1` = second pane (Files), and so on. Values in
 * between mean the user is mid-swipe, which lets the navbar highlight track the
 * gesture continuously instead of jumping once the snap completes.
 *
 * Kept in a module-level signal because the pager (`MobileTabShell`) and the
 * navbar (`MobileNavbar`) are siblings, not parent/child.
 */
const [pagerPosition, setPagerPosition] = createSignal(0);

export { MOBILE_TABS, pageFromPath, indexFromPath, pagerPosition, setPagerPosition };
export type { MobileTab };
