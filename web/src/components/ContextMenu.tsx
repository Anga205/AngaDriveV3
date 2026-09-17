import { Component, JSX, Show, createContext, createSignal, onCleanup, onMount, useContext } from "solid-js";
import ChevronRight from "lucide-solid/icons/chevron-right";

interface MenuContextValue {
    close: () => void;
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
}

/**
 * A reusable, headless-ish context menu. Renders a trigger (via render prop)
 * and an anchored dropdown panel. Closes on outside click or Escape. Any
 * ContextMenuItem inside automatically closes the whole menu when clicked.
 */
const ContextMenu: Component<ContextMenuProps> = (props) => {
    const [open, setOpen] = createSignal(false);
    let rootRef: HTMLDivElement | undefined;

    const toggle = () => setOpen((o) => !o);
    const close = () => setOpen(false);

    const handleClickOutside = (event: MouseEvent) => {
        if (rootRef && !rootRef.contains(event.target as Node)) {
            setOpen(false);
        }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
        if (event.key === "Escape") {
            event.preventDefault();
            setOpen(false);
        }
    };

    onMount(() => {
        document.addEventListener("mousedown", handleClickOutside);
        document.addEventListener("keydown", handleKeyDown);
    });
    onCleanup(() => {
        document.removeEventListener("mousedown", handleClickOutside);
        document.removeEventListener("keydown", handleKeyDown);
    });

    return (
        <MenuContext.Provider value={{ close }}>
            <div class="relative" ref={rootRef}>
                {props.trigger({ open, toggle, close })}
                <Show when={open()}>
                    <div
                        class={`absolute top-full z-20 mt-1 min-w-44 rounded-lg border border-neutral-700 bg-neutral-800 p-1 shadow-xl ${props.align === "left" ? "left-0" : "right-0"} ${props.menuClass || ""}`}
                        onClick={(e) => e.stopPropagation()}
                    >
                        {props.children}
                    </div>
                </Show>
            </div>
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
            class={`flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-neutral-100 hover:bg-neutral-700 ${props.disabled ? "cursor-not-allowed opacity-50" : ""} ${props.class || ""}`}
            onClick={(e) => {
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
 * A nested submenu inside a ContextMenu. Opens on hover (or click) and renders
 * its items in a panel to the right. Items inside still close the whole menu.
 */
const ContextMenuSubmenu: Component<ContextMenuSubmenuProps> = (props) => {
    const [subOpen, setSubOpen] = createSignal(false);
    return (
        <div
            class="relative"
            onMouseEnter={() => setSubOpen(true)}
            onMouseLeave={() => setSubOpen(false)}
        >
            <button
                type="button"
                class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-neutral-100 hover:bg-neutral-700"
                onClick={(e) => {
                    e.stopPropagation();
                    setSubOpen((o) => !o);
                }}
            >
                {props.icon}
                <span class="flex-1">{props.label}</span>
                <ChevronRight class="h-4 w-4 text-neutral-400" />
            </button>
            <Show when={subOpen()}>
                <div
                    class={`absolute left-full top-0 z-20 ml-1 min-w-44 rounded-lg border border-neutral-700 bg-neutral-800 p-1 shadow-xl ${props.menuClass || ""}`}
                    onClick={(e) => e.stopPropagation()}
                >
                    {props.children}
                </div>
            </Show>
        </div>
    );
};

export { ContextMenu, ContextMenuItem, ContextMenuSubmenu, useMenuContext };
