import { For, useContext } from 'solid-js';
import { DesktopTemplate } from '@/components/Template';
import { AppContext } from '@/Context';
import CollectionCard from '@/components/CollectionCard';
import Popup from '../shared/components/Popup';
import CollectionsError from '../shared/components/CollectionsError';

const DesktopCollections = () => {
    const ctx = useContext(AppContext)!;
    const { userCollections } = ctx;
    return (
        <DesktopTemplate CurrentPage='Collections'>
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh] space-y-10">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh]">My Collections</p>
                    <Popup/>
                </div>
                <div class={`w-full ${userCollections().size === 0 ? 'h-full' : 'max-h-full'} flex justify-center flex-wrap gap-8 overflow-y-auto pt-10 custom-scrollbar`}>
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
                </div>
            </div>
        </DesktopTemplate>
    )
}

export default DesktopCollections;