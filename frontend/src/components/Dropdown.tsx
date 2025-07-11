import { Component, For, createSignal, onCleanup, onMount } from 'solid-js';
import { CollectionCardData } from '../library/types';

interface DropdownProps {
    options: CollectionCardData[];
    selected: string[];
    onChange: (selected: string[]) => void;
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

    return (
        <div class="relative" ref={dropdownRef}>
            <button onClick={toggleDropdown} class="w-full p-2 rounded-lg bg-neutral-700 text-white focus:outline-none focus:ring-2 focus:ring-green-500 flex justify-between items-center">
                <span>Select Folders</span>
                <svg class={`w-4 h-4 transition-transform ${isOpen() ? 'transform rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
            </button>
            {isOpen() && (
                <div class="absolute z-10 w-full mt-1 bg-neutral-700 rounded-lg shadow-lg max-h-60 overflow-y-auto">
                    <ul>
                        <For each={props.options}>
                            {(option) => (
                                <li class="p-2 hover:bg-neutral-600 cursor-pointer text-white">
                                    <label class="flex items-center space-x-2">
                                        <input
                                            type="checkbox"
                                            class="form-checkbox h-5 w-5 text-green-600 bg-gray-800 border-gray-600 rounded focus:ring-green-500"
                                            checked={props.selected.includes(option.id)}
                                            onChange={(e) => handleCheckboxChange(option.id, e.currentTarget.checked)}
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