import { Accessor, For, useContext } from 'solid-js';
import Navbar from '@/components/Navbar';
import { AppContext } from '@/Context';
import CollectionCard from '@/components/CollectionCard';
import Popup from '../shared/components/Popup';
import CollectionsError from '../shared/components/CollectionsError';
import Search from 'lucide-solid/icons/search';
import type { CollectionCardData } from '@/library/types';

const MobileCollections = (props: { searchQuery: Accessor<string>; setSearch: (value: string) => void; filterCollections: ((collections: CollectionCardData[]) => CollectionCardData[]) | undefined }) => {
    const ctx = useContext(AppContext)!;
    const { userCollections } = ctx;
    const collections = () => {
        const sorted = [...userCollections()].sort((a, b) => {
            const cardA = ctx.knownCollectionCards()[a];
            const cardB = ctx.knownCollectionCards()[b];
            if (!cardA || !cardB) return 0;
            const tsCompare = cardB.timestamp - cardA.timestamp;
            if (tsCompare !== 0) return tsCompare;
            return a.localeCompare(b);
        }).map((id) => ctx.knownCollectionCards()[id]).filter((collection): collection is CollectionCardData => !!collection);
        return props.filterCollections ? props.filterCollections(sorted) : sorted;
    };
    return (
        <div class="flex flex-col w-full max-h-screen h-screen bg-black">
            <Navbar CurrentPage="Collections" Type="mobile"/>
            <div class="h-[6vh]"/>
            <p class="text-white font-black text-[4vh] px-3">My&nbsp;Collections</p>
            <div class="flex flex-col gap-3 px-3">
                <div class="relative w-full">
                    <input
                        class="w-full bg-neutral-900 placeholder-neutral-500 text-neutral-200 rounded-lg px-3 py-2 pr-10 border border-neutral-800 focus:outline-none"
                        placeholder="Search collections by name"
                        value={props.searchQuery()}
                        onInput={(e) => props.setSearch(e.currentTarget.value)}
                    />
                    <div class="absolute right-2 top-2 text-neutral-400">
                        <Search class="w-5 h-5" />
                    </div>
                </div>
                <div class="flex justify-end">
                    <Popup/>
                </div>
            </div>
            <div class="w-full px-4 mt-4 max-h-full h-full flex flex-wrap space-y-4 space-x-4 justify-center overflow-y-auto">
                <For each={collections()} fallback={props.searchQuery() ? <p class="text-neutral-500 pt-10">No collections match your search.</p> : <CollectionsError />}>
                    {(collection) => <CollectionCard collection={collection} />}
                </For>
                <div class="w-full h-[2vh]"/>
            </div>
        </div>
    )
}

export default MobileCollections;