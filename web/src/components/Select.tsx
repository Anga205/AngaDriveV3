import { Component, Show, createMemo, createSignal, onCleanup, onMount } from "solid-js";

export type SelectOption = { id: string; name: string };

interface SelectProps {
  options: SelectOption[];
  selected: string[]; // for compatibility with existing API, we treat it as single select and use the first value
  onChange: (ids: string[]) => void;
  placeholderText?: string;
  disabled?: boolean;
  class?: string;
  buttonClass?: string;
  listClass?: string;
}

// A small, dependency-free, accessible-ish select primitive for Solid.
// - Single-select but returns [id] to match existing Dropdown API
// - Keyboard: ArrowUp/Down navigate, Enter/Space select, Esc close
// - Click outside to close
// - ARIA roles: combobox + listbox + option
const Select: Component<SelectProps> = (props) => {
  const [open, setOpen] = createSignal(false);
  const [activeIndex, setActiveIndex] = createSignal<number>(-1);
  let buttonRef: HTMLButtonElement | undefined;
  let listRef: HTMLUListElement | undefined;

  const selectedId = createMemo(() => props.selected?.[0]);
  const selectedIndex = createMemo(() => props.options.findIndex(o => o.id === selectedId()));
  const selectedLabel = createMemo(() => props.options.find(o => o.id === selectedId())?.name);

  const setNextActive = (delta: number) => {
    const opts = props.options;
    if (!opts.length) return;
    let idx = activeIndex();
    if (idx < 0) idx = Math.max(selectedIndex(), 0);
    let next = (idx + delta + opts.length) % opts.length;
    setActiveIndex(next);
    // ensure visible
    queueMicrotask(() => {
      const el = listRef?.querySelector<HTMLElement>(`[data-index="${next}"]`);
      el?.scrollIntoView({ block: "nearest" });
    });
  };

  const openList = () => {
    if (props.disabled) return;
    setOpen(true);
    // set initial active item
    const idx = selectedIndex();
    setActiveIndex(idx >= 0 ? idx : 0);
  };
  const closeList = () => setOpen(false);

  const chooseIndex = (idx: number) => {
    const opt = props.options[idx];
    if (!opt) return;
    props.onChange([opt.id]);
    closeList();
    buttonRef?.focus();
  };

  const onKeyDown = (e: KeyboardEvent) => {
    if (!open()) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        openList();
      }
      return;
    }
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setNextActive(1);
        break;
      case "ArrowUp":
        e.preventDefault();
        setNextActive(-1);
        break;
      case "Enter":
      case " ":
        e.preventDefault();
        chooseIndex(activeIndex() >= 0 ? activeIndex() : Math.max(selectedIndex(), 0));
        break;
      case "Escape":
        e.preventDefault();
        closeList();
        buttonRef?.focus();
        break;
      case "Tab":
        // allow default tabbing but close
        closeList();
        break;
    }
  };

  // click outside to close
  const onDocMouseDown = (e: MouseEvent) => {
    const t = e.target as Node;
    if (buttonRef?.contains(t)) return;
    if (listRef?.contains(t)) return;
    setOpen(false);
  };

  onMount(() => {
    document.addEventListener("mousedown", onDocMouseDown);
    document.addEventListener("keydown", onKeyDown as any);
  });
  onCleanup(() => {
    document.removeEventListener("mousedown", onDocMouseDown);
    document.removeEventListener("keydown", onKeyDown as any);
  });

  // style helpers
  const btnClasses = () =>
    props.buttonClass ??
    "w-full min-w-[230px] inline-flex items-center justify-between gap-2 rounded-md border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-white hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-600 disabled:opacity-50";
  const listClasses = () =>
    props.listClass ??
    "absolute z-50 mt-1 w-full max-h-60 overflow-auto rounded-md border border-gray-700 bg-gray-900 p-1 shadow-lg";

  return (
    <div class={props.class} style={{ position: "relative" }}>
      <button
        ref={buttonRef}
        type="button"
        class={btnClasses()}
        aria-haspopup="listbox"
        aria-expanded={open() ? "true" : "false"}
        aria-controls="select-listbox"
        onClick={() => (open() ? closeList() : openList())}
        disabled={props.disabled}
      >
        <span class="truncate">
          {selectedLabel() ?? props.placeholderText ?? "Select"}
        </span>
        <svg
          class={`h-4 w-4 transition-transform ${open() ? "rotate-180" : "rotate-0"}`}
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.24a.75.75 0 01-1.06 0L5.25 8.29a.75.75 0 01-.02-1.08z" clip-rule="evenodd" />
        </svg>
      </button>

      <Show when={open()}>
        <ul
          ref={listRef}
          id="select-listbox"
          role="listbox"
          class={listClasses()}
          tabindex={-1 as unknown as string}
        >
          {props.options.map((opt, idx) => {
            const isActive = () => activeIndex() === idx;
            const isSelected = () => selectedId() === opt.id;
            return (
              <li
                role="option"
                aria-selected={isSelected() ? "true" : "false"}
                data-index={idx}
                class={`flex cursor-pointer select-none items-center rounded px-2 py-1 text-sm text-white outline-none ${
                  isActive() ? "bg-blue-600/30" : isSelected() ? "bg-gray-800" : "hover:bg-gray-800"
                }`}
                onMouseEnter={() => setActiveIndex(idx)}
                onMouseDown={(e) => {
                  // Prevent button blur closing before click handler runs
                  e.preventDefault();
                }}
                onClick={() => chooseIndex(idx)}
              >
                <span class="truncate">{opt.name}</span>
                <Show when={isSelected()}>
                  <svg class="ml-auto h-4 w-4 text-blue-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
                    <path fill-rule="evenodd" d="M16.704 5.29a1 1 0 010 1.42l-7.25 7.25a1 1 0 01-1.42 0l-3.25-3.25a1 1 0 111.42-1.42l2.54 2.54 6.54-6.54a1 1 0 011.42 0z" clip-rule="evenodd" />
                  </svg>
                </Show>
              </li>
            );
          })}
        </ul>
      </Show>
    </div>
  );
};

export default Select;
