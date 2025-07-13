import { Component, For, createSignal, onCleanup, onMount } from 'solid-js';

interface DropdownOption {
    id: string;
    name: string;
}

interface DropdownProps {
    options: DropdownOption[];
    selected: string[];
    onChange: (selected: string[]) => void;
    placeholderText?: string;
}

const Dropdown: Component<DropdownProps> = (props) => {
    const [isOpen, setIsOpen] = createSignal(false);
    let dropdownRef: HTMLDivElement | undefined;

    const toggleDropdown = () => setIsOpen(!isOpen());

    const handleCheckboxChange = (id: string, checked: boolean) => {
        if (checked) {
            props.onChange([...props.selected, id]);
        } else {
            props.onChange(props.selected.filter(item => item !== id));
        }
    };

    const handleUnselect = (id: string) => {
        props.onChange(props.selected.filter(item => item !== id));
    };

    const handleClickOutside = (event: MouseEvent) => {
        if (dropdownRef && !dropdownRef.contains(event.target as Node)) {
            setIsOpen(false);
        }
    };

    onMount(() => {
        document.addEventListener('mousedown', handleClickOutside);
    });

    onCleanup(() => {
        document.removeEventListener('mousedown', handleClickOutside);
    });

    const selectedOptions = () => props.options.filter(option => props.selected.includes(option.id));

    return (
        <div class="relative" ref={dropdownRef}>
            <div onClick={toggleDropdown} class="w-full p-2 rounded-lg bg-neutral-700 text-white focus-within:ring-2 focus-within:ring-green-500 flex justify-between items-center cursor-pointer">
                <div class="flex flex-wrap gap-2 items-center flex-grow">
                    {props.selected.length === 0 ? (
                        <span class="text-gray-400">{props.placeholderText || "Select items"}</span>
                    ) : (
                        <For each={selectedOptions()}>
                            {(option) => (
                                <span class="flex items-center bg-neutral-600 text-white text-sm font-medium px-2.5 py-1 rounded-md">
                                    {option.name}
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation(); // Prevent dropdown from toggling
                                            handleUnselect(option.id);
                                        }}
                                        class="ml-2 text-gray-400 hover:text-white focus:outline-none"
                                    >
                                        &times;
                                    </button>
                                </span>
                            )}
                        </For>
                    )}
                </div>
                <svg class={`w-4 h-4 transition-transform flex-shrink-0 ${isOpen() ? 'transform rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
            </div>
            {isOpen() && (
                <div class="absolute z-10 w-full mt-1 bg-neutral-700 rounded-lg shadow-lg max-h-60 overflow-y-auto">
                    <ul>
                        <For each={props.options}>
                            {(option) => (
                                <li
                                    class="p-2 hover:bg-neutral-600 cursor-pointer text-white"
                                    onClick={() => handleCheckboxChange(option.id, !props.selected.includes(option.id))}
                                >
                                    <label class="flex items-center space-x-2 pointer-events-none">
                                        <input
                                            type="checkbox"
                                            class="form-checkbox h-5 w-5 text-green-600 bg-gray-800 border-gray-600 rounded focus:ring-green-500"
                                            checked={props.selected.includes(option.id)}
                                        />
                                        <span>{option.name}</span>
                                    </label>
                                </li>
                            )}
                        </For>
                    </ul>
                </div>
            )}
        </div>
    );
};

export default Dropdown;