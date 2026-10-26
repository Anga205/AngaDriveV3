import { Component, createSignal, onMount, onCleanup, useContext } from "solid-js";
import { useWebSocket } from "@/Websockets";
import { AppContext } from "@/Context";
import { fetchFilesAndCollections, handleLogout } from "@/library/functions";
import { useIsMobile } from "@/library/useIsMobile";
import AccountManager from "./Desktop/AccountManager";
import MobileAccountManager from "./Mobile/MobileAccountManager";
import { LoginScreen } from "./shared/components/LoginRegister";

const Account: Component = () => {
    const [isLoggedIn, setIsLoggedIn] = createSignal(false);
    const isMobile = useIsMobile();
    const ctx = useContext(AppContext)!;

    const currentSocket = useWebSocket();

    const handleLoginSuccess = () => {
        setIsLoggedIn(true);
        fetchFilesAndCollections(currentSocket.socket()!);
    };

    const syncLoginState = () => {
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");
        setIsLoggedIn(!!(storedEmail && storedPassword));
    };

    onMount(() => {
        syncLoginState();
        window.addEventListener('storage', syncLoginState);
        window.addEventListener('auth-changed', syncLoginState);
    });

    onCleanup(() => {
        window.removeEventListener('storage', syncLoginState);
        window.removeEventListener('auth-changed', syncLoginState);
    });

    return (
        <>
            {isLoggedIn() ?
                (isMobile() ? <MobileAccountManager logout={() => handleLogout(setIsLoggedIn, ctx)} /> : <AccountManager logout={() => handleLogout(setIsLoggedIn, ctx)} />)
                : <LoginScreen onLoginSuccess={handleLoginSuccess} isMobile={isMobile()} />
            }
        </>
    );
}

export default Account;