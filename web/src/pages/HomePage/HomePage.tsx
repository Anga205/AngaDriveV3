import { Component, onCleanup, createSignal, createEffect } from "solid-js";
import { useWebSocket } from "@/Websockets";
import type { GraphData, IncomingData, SysInfo } from "@/library/types";
import { useIsMobile } from "@/library/useIsMobile";
import DesktopHome from "./Desktop/DesktopHome";
import MobileHome from "./Mobile/MobileHome";

const HomePage: Component = () => {
    const isMobile = useIsMobile();
    const [userCount, setUserCount] = createSignal(0);
    const [filesHosted, setFilesHosted] = createSignal(0);

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
                // Remove listeners from the *old* socket (if there was one)
                if (currentSocket) {
                    currentSocket.removeEventListener('message', messageHandler);
                    currentSocket.removeEventListener('open', socketOpenHandler);

                    if (currentSocket.readyState === WebSocket.OPEN) {
                        currentSocket.send(JSON.stringify({ type: 'enable_homepage_updates', data: false }));
                    }
                }
            });
        }
    });

    return (
        <>
            {
            isMobile() ? (
                <MobileHome cpudata={systemInformation()!.cpu} ramdata={systemInformation()!.ram} siteActivity={siteActivity} spaceUsed={spaceUsed} userCount={userCount} filesHosted={filesHosted} />
            ) : (
                <DesktopHome cpudata={systemInformation()!.cpu} ramdata={systemInformation()!.ram} siteActivity={siteActivity} spaceUsed={spaceUsed} userCount={userCount} filesHosted={filesHosted} />
            )
            }
        </>
    );
}

export default HomePage;