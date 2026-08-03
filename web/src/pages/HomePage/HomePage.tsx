import { Component, onMount, onCleanup, createSignal, createEffect } from "solid-js";
import { Toaster } from 'solid-toast';
import { useWebSocket } from "@/Websockets";
import type { GraphData, IncomingData, SysInfo } from "@/library/types";
import DesktopHome from "./Desktop/DesktopHome";
import MobileHome from "./Mobile/MobileHome";

const HomePage: Component = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 640);
    const [userCount, setUserCount] = createSignal(0);
    const [filesHosted, setFilesHosted] = createSignal(0);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 640);
    };
    const [systemInformation, setSystemInformation] = createSignal<SysInfo>({
        ram: {
            total_ram: 1,
            used_ram: 0,
            free_ram: 1,
            ram_percent_used: 0,
        },
        cpu: {
            cpu_model_name: '',
            cpu_usage: 0,
        },
    });

    const [spaceUsed, setSpaceUsed] = createSignal<GraphData>({
        x_axis: Array.from({ length: 7 }, (_, i) => {
            const date = new Date();
            date.setDate(date.getDate() - (6 - i));
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }),
        y_axis: [0, 0, 0, 0, 0, 0, 0],
        label: 'Space Used',
        begin_at_zero: false,
    });

    const [siteActivity, setSiteActivity] = createSignal<GraphData>({
        x_axis: Array.from({ length: 7 }, (_, i) => {
            const date = new Date();
            date.setDate(date.getDate() - (6 - i));
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }),
        y_axis: [0, 0, 0, 0, 0, 0, 0],
        label: 'Database Reads',
        begin_at_zero: true,
    })

    const { socket: getSocket } = useWebSocket();

    onMount(() => {
        window.addEventListener('resize', handleResize);
    });

    let currentSocket: WebSocket | null = null; // Keep track of the current socket

    const messageHandler = (event: MessageEvent) => {
        try {
            const data = JSON.parse(event.data) as IncomingData;
            if (data.type === 'system_information') {
                setSystemInformation(data.data as SysInfo);
            } else if (data.type === 'graph_data') {
                const graphData = data.data as GraphData;
                if (graphData.label === 'Space Used') {
                    graphData.y_axis = graphData.y_axis.map(bytes => (bytes / (1024 ** 3)));
                                        graphData.label = 'Space Used (GB)';
                    setSpaceUsed(graphData);
                } else if (graphData.label === 'Site Activity' || graphData.label === 'Database Reads') {
                    setSiteActivity(graphData);
                }
            } else if (data.type === 'user_count') {
                setUserCount(data.data as number);
            }
            else if (data.type === 'files_hosted_count') {
                setFilesHosted(data.data as number);
            }
        } catch (error) {
            if (import.meta.env.DEV) {
                console.error('Failed to parse WebSocket message:', error);
            }
        }
    };

    const openHandler = (updatedSocket: WebSocket) => {
        if (updatedSocket.readyState === WebSocket.OPEN) {
            console.log("HomePage.tsx: Socket open, enabling updates.");
            updatedSocket.send(JSON.stringify({ type: 'enable_homepage_updates', data: true }));
        }
    };

    createEffect(() => {
        const newSocket = getSocket(); // Get the latest socket from the signal

        if (newSocket) {



            // Add listeners to the new socket
            newSocket.addEventListener('message', messageHandler);
            const socketOpenHandler = () => openHandler(newSocket);
            newSocket.addEventListener('open', socketOpenHandler);

            if (newSocket.readyState === WebSocket.OPEN) {
                openHandler(newSocket);
            }

            // Cleanup function (called when the effect re-runs or the component unmounts)

            currentSocket = newSocket; // Update the current socket
            onCleanup(() => {
                console.log("HomePage.tsx: Cleaning up socket listeners.");

                // Remove listeners from the *old* socket (if there was one)
                if (currentSocket) {
                    currentSocket.removeEventListener('message', messageHandler);
                    currentSocket.removeEventListener('open', socketOpenHandler);

                    if (currentSocket.readyState === WebSocket.OPEN) {
                        console.log("HomePage.tsx: Disabling updates for old/unmounting socket.");
                        currentSocket.send(JSON.stringify({ type: 'enable_homepage_updates', data: false }));
                    }
                }
            });
        }
    });

    onCleanup(() => {
        window.removeEventListener('resize', handleResize);
    });

    return (
        <>
            <title>HomePage | DriveV3</title>
            {
            isMobile() ? (
                <MobileHome cpudata={systemInformation()!.cpu} ramdata={systemInformation()!.ram} siteActivity={siteActivity} spaceUsed={spaceUsed} userCount={userCount} filesHosted={filesHosted} />
            ) : (
                <DesktopHome cpudata={systemInformation()!.cpu} ramdata={systemInformation()!.ram} siteActivity={siteActivity} spaceUsed={spaceUsed} userCount={userCount} filesHosted={filesHosted} />
            )
            }
            <Toaster
            position="top-center"
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

export default HomePage;