import { For, useContext } from 'solid-js';
import Navbar from '@/components/Navbar';
import { AppContext } from '@/Context';
import CollectionCard from '@/components/CollectionCard';
import Popup from '../shared/components/Popup';
import CollectionsError from '../shared/components/CollectionsError';

const MobileCollections = () => {
    const ctx = useContext(AppContext)!;
    const { userCollections } = ctx;
    return (
        <div class="flex flex-col w-full max-h-screen h-screen bg-black">
            <Navbar CurrentPage="Collections" Type="mobile"/>
            <div class="h-[6vh]"/>
            <p class="text-white font-black text-[4vh] px-3">My&nbsp;Collections</p>
            <div class="flex justify-end px-3">
                <Popup/>
            </div>
            <div class="w-full px-4 mt-4 max-h-full h-full flex flex-wrap space-y-4 space-x-4 justify-center overflow-y-auto">
                <For each={[...userCollections()].sort((a, b) => {
                    const cardA = ctx.knownCollectionCards()[a];
                    const cardB = ctx.knownCollectionCards()[b];
                    if (!cardA || !cardB) return 0;
                    const tsCompare = cardB.timestamp - cardA.timestamp;
                    if (tsCompare !== 0) return tsCompare;
                    return a.localeCompare(b);
                })} fallback={<CollectionsError />}>
                    {(collection: string) => (
                        <CollectionCard collection={ctx.knownCollectionCards()[collection]} />
                    )}
                </For>
                <div class="w-full h-[2vh]"/>
            </div>
        </div>
    )
}

export default MobileCollections;