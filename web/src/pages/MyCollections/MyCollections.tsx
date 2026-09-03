import { createMemo, createSignal } from 'solid-js';
import { Toaster } from 'solid-toast';
import type { CollectionCardData } from '@/library/types';
import DesktopCollections from './Desktop/DesktopCollections';
import MobileCollections from './Mobile/MobileCollections';

const MyCollections = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };
    window.addEventListener('resize', handleResize);

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
            <title>My Collections | DriveV3</title>
            {isMobile() ?
                <MobileCollections searchQuery={searchQuery} setSearch={setSearchQuery} filterCollections={filteredCollections()} /> :
                <DesktopCollections searchQuery={searchQuery} setSearch={setSearchQuery} filterCollections={filteredCollections()} />}
            <Toaster
            position="bottom-right"
            gutter={8}
            containerClassName=""
            containerStyle={{}}
            toastOptions={{
                className: '',
                duration: 2000,
                style: {
                background: '#363636',
                color: '#fff',
                },
            }}
            />
        </>
    )
}

export default MyCollections