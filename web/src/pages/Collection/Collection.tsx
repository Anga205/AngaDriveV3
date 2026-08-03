import { Component, createSignal, onMount, onCleanup } from "solid-js";
import { Toaster } from "solid-toast";
import CollectionPageDesktop from "./Desktop/CollectionPageDesktop";
import CollectionPageMobile from "./Mobile/CollectionPageMobile";

const CollectionPage: Component = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };

    onMount(() => {
        window.addEventListener('resize', handleResize);
    });

    onCleanup(() => {
        window.removeEventListener('resize', handleResize);
    });

    return (
        <>
            <title>Collection | DriveV3</title>
            {isMobile() ? <CollectionPageMobile /> : <CollectionPageDesktop />}
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
    );
}

export default CollectionPage;