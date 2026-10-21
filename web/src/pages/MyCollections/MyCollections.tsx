import { createMemo, createSignal } from 'solid-js';
import type { CollectionCardData } from '@/library/types';
import { useIsMobile } from '@/library/useIsMobile';
import DesktopCollections from './Desktop/DesktopCollections';
import MobileCollections from './Mobile/MobileCollections';

const MyCollections = () => {
    const isMobile = useIsMobile();

    const [searchQuery, setSearchQuery] = createSignal('');
    const filteredCollections = createMemo(() => {
        const query = searchQuery().trim().toLowerCase();
        if (!query) return undefined;
        return (collections: CollectionCardData[]) => collections.filter((collection) =>
            collection.name.toLowerCase().includes(query) || collection.id.toLowerCase().includes(query)
        );
    });

    return (
        <>
            {isMobile() ?
                <MobileCollections searchQuery={searchQuery} setSearch={setSearchQuery} filterCollections={filteredCollections()} /> :
                <DesktopCollections searchQuery={searchQuery} setSearch={setSearchQuery} filterCollections={filteredCollections()} />}
        </>
    )
}

export default MyCollections