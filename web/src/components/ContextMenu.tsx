import {
    Component, JSX, Show, createContext, createEffect, createSignal, createUniqueId,
    onCleanup, onMount, useContext, type Accessor
} from "solid-js";
import { Portal } from "solid-js/web";
import ChevronRight from "lucide-solid/icons/chevron-right";
import {
    buildAimTriangle, clamp, computePanelPosition, computeSubmenuPosition, pointInTriangle,
    type PanelPosition, type Point, type SubmenuPosition
} from "../library/menuGeometry";

const MENU_GAP = 4;
const MENU_MARGIN = 8;
const AIM_LOOKAHEAD_MS = 200;
const AIM_EXPIRY_MS = 350;
const CLOSE_DELAY_MS = 120;

/** True only for devices with a real hover-capable pointer (mouse/trackpad). */
function canHover(): boolean {
    return typeof window !== "undefined" && window.matchMedia("(hover: hover) and (pointer: fine)").matches;
}

interface MenuContextValue {
    close: () => void;
    focusTrigger: () => void;
}

const MenuContext = createContext<MenuContextValue>();

function useMenuContext() {
    return useContext(MenuContext);
}

interface ContextMenuProps {
    /** Render-prop for the trigger element. Receives open state and controls. */
    trigger: (state: { open: () => boolean; toggle: () => void; close: () => void }) => JSX.Element;
    /** Menu items (ContextMenuItem / ContextMenuSubmenu). */
    children: JSX.Element;
    /** Extra classes for the dropdown panel. */
    menuClass?: string;
    /** Horizontal alignment of the dropdown relative to the trigger. */
    align?: "left" | "right";
    /** Notifies the owner when the menu opens or closes. */
    onOpenChange?: (open: boolean) => void;
}

interface PositioningOptions {
    open: Accessor<boolean>;
    getAnchor: () => HTMLElement | undefined;
    getPanel: () => HTMLElement | undefined;
    mode: "menu" | "submenu";
    align?: Accessor<"left" | "right" | undefined>;
}

/**
 * Shared viewport-aware positioning for menu panels. Measures the anchor and
 * the (portaled) panel, computes clamped fixed coordinates, and keeps them in
 * sync while the menu is open. Scroll is listened for in the capture phase so
 * nested scroll containers (e.g. the mobile tab pager) are covered too.
 */
function createMenuPositioning(options: PositioningOptions) {
    const [position, setPosition] = createSignal<PanelPosition | SubmenuPosition>({
        left: 0, top: 0, placement: "bottom"
    });
    const [ready, setReady] = createSignal(false);
    let frame = 0;

    const measure = () => {
        const anchor = options.getAnchor();
        const panel = options.getPanel();
        if (!anchor || !panel) return;
        const anchorRect = anchor.getBoundingClientRect();
        const panelRect = panel.getBoundingClientRect();
        const size = { width: panelRect.width, height: panelRect.height };
        const next = options.mode === "menu"
            ? computePanelPosition(anchorRect, size, options.align?.() ?? "right", MENU_GAP, MENU_MARGIN)
            : computeSubmenuPosition(anchorRect, size, MENU_GAP, MENU_MARGIN);
        setPosition(next);
        setReady(true);
    };

    const schedule = () => {
        if (frame) cancelAnimationFrame(frame);
        frame = requestAnimationFrame(() => {
            frame = 0;
            measure();
        });
    };

    createEffect(() => {
        if (options.open()) schedule();
        else setReady(false);
    });

    onMount(() => {
        window.addEventListener("scroll", schedule, true);
        window.addEventListener("resize", schedule);
        window.addEventListener("orientationchange", schedule);
    });
    onCleanup(() => {
        if (frame) cancelAnimationFrame(frame);
        window.removeEventListener("scroll", schedule, true);
        window.removeEventListener("resize", schedule);
        window.removeEventListener("orientationchange", schedule);
    });

    return { position, ready, reposition: schedule };
}

function panelStyle(
    position: PanelPosition | SubmenuPosition,
    ready: boolean
): JSX.CSSProperties {
    return {
        position: "fixed",
        left: `${position.left}px`,
        top: `${position.top}px`,
        "max-height": position.maxHeight !== undefined ? `${position.maxHeight}px` : "none",
        "overflow-y": position.maxHeight !== undefined ? "auto" : "visible",
        visibility: ready ? "visible" : "hidden"
    } as JSX.CSSProperties;
}

function getMenuItems(panel: HTMLElement | undefined): HTMLElement[] {
    if (!panel) return [];
    return Array.from(panel.querySelectorAll<HTMLElement>('[data-menu-item]:not([aria-disabled="true"])'));
}

/** Move DOM focus between enabled items in a menu panel, with wraparound. */
function moveFocus(panel: HTMLElement | undefined, direction: number | "first" | "last"): void {
    const items = getMenuItems(panel);
    if (items.length === 0) return;
    const active = document.activeElement as HTMLElement | null;
    let index = items.indexOf(active as HTMLElement);
    if (direction === "first") index = 0;
    else if (direction === "last") index = items.length - 1;
    else index = index < 0
        ? (direction > 0 ? 0 : items.length - 1)
        : (index + direction + items.length) % items.length;
    items[index]?.focus();
}

/**
 * A reusable, accessible context menu. The panel is portaled to <body> and
 * positioned with fixed coordinates clamped to the viewport, so it can never
 * be clipped by an ancestor's `overflow: hidden` (notably the mobile pager).
 * Closes on outside pointerdown or Escape; items close the whole menu.
 */
const ContextMenu: Component<ContextMenuProps> = (props) => {
    const [open, setOpen] = createSignal(false);
    const panelId = createUniqueId();
    let rootRef: HTMLDivElement | undefined;
    let panelRef: HTMLDivElement | undefined;

    const triggerEl = () => rootRef?.firstElementChild as HTMLElement | undefined;

    const setOpenState = (next: boolean) => {
        setOpen(next);
        props.onOpenChange?.(next);
    };

    const closeMenu = (restoreFocus = true) => {
        if (!open()) return;
        setOpenState(false);
        if (restoreFocus) queueMicrotask(() => triggerEl()?.focus());
    };
    const close = () => closeMenu(true);

    const toggle = () => {
        const next = !open();
        setOpenState(next);
        if (!next) queueMicrotask(() => triggerEl()?.focus());
    };

    const { position, ready } = createMenuPositioning({
        open,
        getAnchor: () => rootRef,
        getPanel: () => panelRef,
        mode: "menu",
        align: () => props.align
    });

    const handlePointerDown = (event: PointerEvent) => {
        const target = event.target as Node;
        if (rootRef?.contains(target)) return;
        if (target instanceof Element && target.closest("[data-menu-panel]")) return;
        closeMenu(false);
    };

    const handleKeyDown = (event: KeyboardEvent) => {
        if (event.key !== "Escape" || event.defaultPrevented) return;
        // Panels handle their own Escape (submenus close only themselves).
        if (event.target instanceof Element && event.target.closest("[data-menu-panel]")) return;
        event.preventDefault();
        close();
    };

    const onPanelKeyDown = (event: KeyboardEvent) => {
        switch (event.key) {
            case "ArrowDown": event.preventDefault(); moveFocus(panelRef, 1); break;
            case "ArrowUp": event.preventDefault(); moveFocus(panelRef, -1); break;
            case "Home": event.preventDefault(); moveFocus(panelRef, "first"); break;
            case "End": event.preventDefault(); moveFocus(panelRef, "last"); break;
            case "Escape": event.preventDefault(); event.stopPropagation(); close(); break;
            case "Tab": closeMenu(false); break;
        }
    };

    createEffect(() => {
        if (!open()) return;
        requestAnimationFrame(() => getMenuItems(panelRef)[0]?.focus());
    });

    onMount(() => {
        document.addEventListener("pointerdown", handlePointerDown);
        document.addEventListener("keydown", handleKeyDown);
    });
    onCleanup(() => {
        document.removeEventListener("pointerdown", handlePointerDown);
        document.removeEventListener("keydown", handleKeyDown);
    });

    return (
        <MenuContext.Provider value={{ close, focusTrigger: () => triggerEl()?.focus() }}>
            <div class="relative" ref={rootRef}>
                {props.trigger({ open, toggle, close })}
            </div>
            <Show when={open()}>
                <Portal>
                    <div
                        ref={panelRef}
                        id={panelId}
                        data-menu-panel
                        role="menu"
                        tabindex={-1}
                        class={`z-60 min-w-56 rounded-lg border border-neutral-700 bg-neutral-800 p-1 shadow-xl outline-none ${props.menuClass || ""}`}
                        style={panelStyle(position(), ready())}
                        onClick={(e) => e.stopPropagation()}
                        onKeyDown={onPanelKeyDown}
                    >
                        {props.children}
                    </div>
                </Portal>
            </Show>
        </MenuContext.Provider>
    );
};

interface ContextMenuItemProps {
    icon?: JSX.Element;
    children: JSX.Element;
    onClick?: (e: MouseEvent) => void;
    class?: string;
    disabled?: boolean;
}

/** A single clickable item inside a ContextMenu. Closes the menu on click. */
const ContextMenuItem: Component<ContextMenuItemProps> = (props) => {
    const menu = useMenuContext();
    return (
        <button
            type="button"
            role="menuitem"
            tabindex={-1}
            data-menu-item
            aria-disabled={props.disabled ? "true" : undefined}
            class={`flex w-full items-center gap-2 whitespace-nowrap rounded-md px-3 py-2 text-left text-sm text-neutral-100 hover:bg-neutral-700 focus-visible:bg-neutral-700 focus:outline-none ${props.disabled ? "cursor-not-allowed opacity-50" : ""} ${props.class || ""}`}
            onClick={(e) => {
                if (props.disabled) return;
                props.onClick?.(e);
                menu?.close();
            }}
            disabled={props.disabled}
        >
            {props.icon}
            <span class="flex-1">{props.children}</span>
        </button>
    );
};

interface ContextMenuSubmenuProps {
    label: JSX.Element;
    icon?: JSX.Element;
    children: JSX.Element;
    menuClass?: string;
}

/**
 * A nested submenu. On hover-capable devices it opens on hover and uses the
 * Menu Aim "safe triangle" so diagonal movement toward the panel keeps it
 * open while movement toward a sibling item closes it. On touch devices it
 * opens on tap only. The panel is portaled and viewport-clamped like the root
 * menu, so it flips to the left near the right edge of the screen.
 */
const ContextMenuSubmenu: Component<ContextMenuSubmenuProps> = (props) => {
    const [subOpen, setSubOpen] = createSignal(false);
    const panelId = createUniqueId();
    let wrapperRef: HTMLDivElement | undefined;
    let triggerRef: HTMLButtonElement | undefined;
    let panelRef: HTMLDivElement | undefined;

    let closeTimer: number | undefined;
    let aimTriangle: [Point, Point, Point] | undefined;
    let aimExpiresAt = 0;
    const samples: { x: number; y: number; t: number }[] = [];

    const { position, ready } = createMenuPositioning({
        open: subOpen,
        getAnchor: () => wrapperRef,
        getPanel: () => panelRef,
        mode: "submenu"
    });

    const clearCloseTimer = () => {
        if (closeTimer !== undefined) {
            window.clearTimeout(closeTimer);
            closeTimer = undefined;
        }
    };

    const closeSubmenu = (restoreFocus = false) => {
        clearCloseTimer();
        aimTriangle = undefined;
        setSubOpen(false);
        if (restoreFocus) queueMicrotask(() => triggerRef?.focus());
    };

    const scheduleClose = () => {
        clearCloseTimer();
        closeTimer = window.setTimeout(() => closeSubmenu(false), CLOSE_DELAY_MS);
    };

    const openSubmenu = () => {
        clearCloseTimer();
        aimTriangle = undefined;
        setSubOpen(true);
    };

    const toggleSubmenu = () => subOpen() ? closeSubmenu(false) : openSubmenu();

    const onWrapperPointerMove = (event: PointerEvent) => {
        if (event.pointerType === "touch") return;
        samples.push({ x: event.clientX, y: event.clientY, t: performance.now() });
        if (samples.length > 3) samples.shift();
        if (subOpen()) clearCloseTimer();
    };

    const onWrapperPointerEnter = () => {
        if (!canHover()) return;
        openSubmenu();
    };

    const onWrapperPointerLeave = (event: PointerEvent) => {
        if (!canHover() || !subOpen()) return;
        const exitPoint: Point = { x: event.clientX, y: event.clientY };
        const recent = samples[samples.length - 1];
        const previous = samples[samples.length - 2];
        let velocity: Point = { x: 0, y: 0 };
        if (recent && previous && recent.t > previous.t) {
            const dt = recent.t - previous.t;
            velocity = { x: (recent.x - previous.x) / dt, y: (recent.y - previous.y) / dt };
        }
        const panel = panelRef;
        if (panel) {
            const rect = panel.getBoundingClientRect();
            const nearX = position().placement === "right" ? rect.left : rect.right;
            const nearY = clamp(exitPoint.y, rect.top, rect.bottom);
            aimTriangle = buildAimTriangle(exitPoint, velocity, { x: nearX, y: nearY }, AIM_LOOKAHEAD_MS);
            aimExpiresAt = performance.now() + AIM_EXPIRY_MS;
        } else {
            aimTriangle = undefined;
        }
        scheduleClose();
    };

    const onDocPointerMove = (event: PointerEvent) => {
        if (!subOpen()) return;
        const target = event.target;
        if (panelRef && target instanceof Node && panelRef.contains(target)) {
            clearCloseTimer();
            aimTriangle = undefined;
            return;
        }
        if (wrapperRef && target instanceof Node && wrapperRef.contains(target)) {
            clearCloseTimer();
            return;
        }
        if (closeTimer !== undefined && aimTriangle && performance.now() < aimExpiresAt) {
            const point: Point = { x: event.clientX, y: event.clientY };
            if (pointInTriangle(point, ...aimTriangle)) {
                clearCloseTimer();
                aimTriangle = undefined;
                return;
            }
        }
        // Moving to a different menu item means the user changed their mind.
        if (target instanceof Element && target.closest("[data-menu-item]")) {
            closeSubmenu(false);
        }
    };

    const onPanelPointerEnter = () => {
        clearCloseTimer();
        aimTriangle = undefined;
    };
    const onPanelPointerLeave = () => {
        if (canHover()) scheduleClose();
    };

    const focusFirstItem = () => {
        requestAnimationFrame(() => getMenuItems(panelRef)[0]?.focus());
    };

    const onTriggerKeyDown = (event: KeyboardEvent) => {
        if (event.key === "ArrowRight") {
            event.preventDefault();
            event.stopPropagation();
            openSubmenu();
            focusFirstItem();
        }
    };

    const onPanelKeyDown = (event: KeyboardEvent) => {
        switch (event.key) {
            case "ArrowDown": event.preventDefault(); moveFocus(panelRef, 1); break;
            case "ArrowUp": event.preventDefault(); moveFocus(panelRef, -1); break;
            case "Home": event.preventDefault(); moveFocus(panelRef, "first"); break;
            case "End": event.preventDefault(); moveFocus(panelRef, "last"); break;
            case "ArrowLeft":
                event.preventDefault(); event.stopPropagation(); closeSubmenu(true); break;
            case "Escape":
                event.preventDefault(); event.stopPropagation(); closeSubmenu(true); break;
            case "Tab": closeSubmenu(false); break;
        }
    };

    onMount(() => document.addEventListener("pointermove", onDocPointerMove));
    onCleanup(() => {
        document.removeEventListener("pointermove", onDocPointerMove);
        clearCloseTimer();
    });

    return (
        <div
            ref={wrapperRef}
            class="relative"
            onPointerEnter={onWrapperPointerEnter}
            onPointerLeave={onWrapperPointerLeave}
            onPointerMove={onWrapperPointerMove}
        >
            <button
                ref={triggerRef}
                type="button"
                role="menuitem"
                tabindex={-1}
                data-menu-item
                aria-haspopup="menu"
                aria-expanded={subOpen()}
                aria-controls={panelId}
                class="flex w-full items-center gap-2 whitespace-nowrap rounded-md px-3 py-2 text-left text-sm text-neutral-100 hover:bg-neutral-700 focus-visible:bg-neutral-700 focus:outline-none"
                onClick={(e) => {
                    e.stopPropagation();
                    // On hover devices the pointer already opened the submenu
                    // on enter; clicking must not immediately close it again.
                    if (canHover()) openSubmenu();
                    else toggleSubmenu();
                }}
                onKeyDown={onTriggerKeyDown}
            >
                {props.icon}
                <span class="flex-1">{props.label}</span>
                <ChevronRight class="h-4 w-4 text-neutral-400" />
            </button>
            <Show when={subOpen()}>
                <Portal>
                    <div
                        ref={panelRef}
                        id={panelId}
                        data-menu-panel
                        role="menu"
                        tabindex={-1}
                        class={`z-70 min-w-56 rounded-lg border border-neutral-700 bg-neutral-800 p-1 shadow-xl outline-none ${props.menuClass || ""}`}
                        style={panelStyle(position(), ready())}
                        onClick={(e) => e.stopPropagation()}
                        onKeyDown={onPanelKeyDown}
                        onPointerEnter={onPanelPointerEnter}
                        onPointerLeave={onPanelPointerLeave}
                    >
                        {props.children}
                    </div>
                </Portal>
            </Show>
        </div>
    );
};

export { ContextMenu, ContextMenuItem, ContextMenuSubmenu, useMenuContext };
