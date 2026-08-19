import { Component, createSignal, onMount, onCleanup, useContext } from "solid-js";
import { Toaster } from "solid-toast";
import { useWebSocket } from "@/Websockets";
import { AppContext } from "@/Context";
import { fetchFilesAndCollections, handleLogout } from "@/library/functions";
import AccountManager from "./Desktop/AccountManager";
import MobileAccountManager from "./Mobile/MobileAccountManager";
import { LoginScreen } from "./shared/components/LoginRegister";

const Account: Component = () => {
    const [isLoggedIn, setIsLoggedIn] = createSignal(false);
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);
    const ctx = useContext(AppContext)!;

    const currentSocket = useWebSocket();

    const handleLoginSuccess = () => {
        setIsLoggedIn(true);
        fetchFilesAndCollections(currentSocket.socket()!);
    };

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };

    onMount(() => {
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");
        setIsLoggedIn(!!(storedEmail && storedPassword));
        window.addEventListener('resize', handleResize);
    });

    onCleanup(() => {
        window.removeEventListener('resize', handleResize);
    });

    window.addEventListener('storage', () => {
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");
        setIsLoggedIn(!!(storedEmail && storedPassword));
    });

    return (
        <>
            <title>Account | DriveV3</title>
            {isLoggedIn() ?
                (isMobile() ? <MobileAccountManager logout={() => handleLogout(setIsLoggedIn, ctx)} /> : <AccountManager logout={() => handleLogout(setIsLoggedIn, ctx)} />)
                : <LoginScreen onLoginSuccess={handleLoginSuccess} isMobile={isMobile()} />
            }
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

export default Account;