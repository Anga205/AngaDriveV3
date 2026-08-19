import { createSignal } from 'solid-js';
import { Toaster } from 'solid-toast';
import DesktopCollections from './Desktop/DesktopCollections';
import MobileCollections from './Mobile/MobileCollections';

const MyCollections = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };
    window.addEventListener('resize', handleResize);

    return (
        <>
            <title>My Collections | DriveV3</title>
            {isMobile() ? <MobileCollections/> : <DesktopCollections/>}
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