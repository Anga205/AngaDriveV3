import { Accessor, Component, createSignal, onMount } from "solid-js";
import Dialog from '@corvu/dialog'; // Or import { Root, Trigger, ... } from '@corvu/dialog'
import { DesktopTemplate } from "../components/Template";
import { toast, Toaster } from "solid-toast";
import { useWebSocket } from "../Websockets";
import bcrypt from "bcryptjs";

const isEmailValid = (email: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
};


const isPasswordValid = (password: string): boolean => {
    return password.length >= 3 && password.length <= 64;
};

const LoginCard: Component<{ onSignUpClick: () => void; onLoginSuccess: () => void }> = (props) => {
    const [email, setEmail] = createSignal("");
    const [password, setPassword] = createSignal("");
    const { socket: getSocket, status } = useWebSocket();

    const isFormValid = () => isEmailValid(email()) && isPasswordValid(password());

    const handleLogin = (e: Event) => {
        e.preventDefault();
        if (!email() || !password()) {
            toast.error(
            !email() && !password()
                ? "Email and password cannot be empty."
                : !email()
                ? "Email cannot be empty."
                : "Password cannot be empty."
            );
            return;
        }
        if (!isEmailValid(email())) {
            toast.error("Please enter a valid email address.");
            return;
        }
        if (!isPasswordValid(password())) {
            toast.error("Password must be between 3 and 64 characters.");
            return;
        }

        const currentSocket = getSocket();

        if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not connected. Please try again later.");
            return;
        }

        let messageHandler: ((event: MessageEvent) => void) | null = null;
        let errorHandler: ((event: Event) => void) | null = null;
        let closeHandler: (() => void) | null = null;

        const cleanupListeners = () => {
            if (messageHandler && currentSocket) currentSocket.removeEventListener('message', messageHandler);
            if (errorHandler && currentSocket) currentSocket.removeEventListener('error', errorHandler);
            if (closeHandler && currentSocket) currentSocket.removeEventListener('close', closeHandler);
            messageHandler = null;
            errorHandler = null;
            closeHandler = null;
        };

        messageHandler = (event: MessageEvent) => {
            try {
                const response = JSON.parse(event.data);
                if (response.type === "login_response") {
                    if (response.data.token) {
                        toast.success("Login successful!");
                        localStorage.setItem("email", email());
                        localStorage.setItem("password", password());
                        localStorage.setItem("display_name", response.data.display_name);
                        props.onLoginSuccess(); // Call the callback on successful login
                    } else {
                        toast.error(
                            response.data.error === "record not found"
                                ? "Account not found"
                                : response.data.error
                                    ? response.data.error.charAt(0).toUpperCase() + response.data.error.slice(1)
                                    : "Login failed. Please try again."
                        );
                    }
                    cleanupListeners();
                }
            } catch (err) {
                console.error("Failed to parse login response:", err);
                toast.error("Received an invalid response from server.");
                cleanupListeners();
            }
        };

        errorHandler = (error: Event) => {
            console.error("WebSocket error during login:", error);
            toast.error("An error occurred during login. Please try again later.");
            cleanupListeners();
        };

        closeHandler = () => {
            console.warn("WebSocket connection closed during login attempt.");
            toast.error("Login failed: Connection lost. Please try again.");
            cleanupListeners();
        };

        currentSocket.addEventListener('message', messageHandler);
        currentSocket.addEventListener('error', errorHandler);
        currentSocket.addEventListener('close', closeHandler);

        currentSocket.send(
            JSON.stringify({
                type: "login",
                data: {
                    email: email(),
                    password: password(),
                },
            })
        );
    };

    return (
        <div class="flex flex-col items-center p-6 border-2 border-gray-500 rounded-lg min-w-[350px] bg-gray-900 shadow-lg overflow-hidden">
            <p class="font-bold text-[3.5vh] mb-6">Login</p>
            <div class="w-full mb-4">
                <p class="text-[1.8vh] mb-1">Enter Email ID:</p>
                <input
                    type="email"
                    class="w-full p-3 rounded bg-gray-800 text-[1.5vh] placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                    placeholder="john.doe@example.com"
                    value={email()}
                    onInput={e => setEmail(e.currentTarget.value)}
                />
            </div>
            <div class="w-full mb-6">
                <p class="text-[1.8vh] mb-1">Enter Password:</p>
                <input
                    type="password"
                    class="w-full p-3 rounded bg-gray-800 text-[1.5vh] placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                    placeholder="Password"
                    value={password()}
                    onInput={e => setPassword(e.currentTarget.value)}
                />
            </div>
            <button
                class={`w-full py-3 mb-4 bg-blue-500 text-white rounded-lg hover:bg-blue-600 ${isFormValid()?'':'cursor-not-allowed opacity-50'}`}
                onClick={handleLogin}
                disabled={status() === "connecting" || status() === "reconnecting"}
            >
                Login
            </button>
            <div class="w-full text-center">
                <span class="text-gray-400 text-[1.5vh]">
                    New to AngaDrive?{" "}
                    <a
                        href="#"
                        class="text-green-400 hover:underline"
                        onClick={e => {
                            e.preventDefault();
                            props.onSignUpClick();
                        }}
                    >
                        Create an account
                    </a>
                </span>
            </div>
        </div>
    );
};

const RegisterCard: Component<{ onLoginClick: () => void; onRegisterSuccess: () => void }> = (props) => {
    const [displayName, setDisplayName] = createSignal("");
    const [email, setEmail] = createSignal("");
    const [password, setPassword] = createSignal("");
    const [confirmPassword, setConfirmPassword] = createSignal("");
    const { socket: getSocket, status } = useWebSocket();
    
    const isFormValid = () => {
        return (
            displayName().length >= 3 &&
            displayName().length <= 64 &&
            isEmailValid(email()) &&
            isPasswordValid(password()) &&
            password() === confirmPassword()
        );
    }
    const handleRegister = (e: Event) => {
        e.preventDefault();
        if (!displayName() || !email() || !password() || !confirmPassword()) {
            toast.error(
                !displayName() && !email() && !password() && !confirmPassword()
                    ? "All fields cannot be empty."
                    : !displayName()
                    ? "Display Name cannot be empty."
                    : !email()
                    ? "Email cannot be empty."
                    : !password()
                    ? "Password cannot be empty."
                    : "Confirm Password cannot be empty."
            );
            return;
        }
        if (displayName().length < 3 || displayName().length > 64) {
            toast.error("Display name must be between 3 and 64 characters.");
            return;
        }
        if (!isEmailValid(email())) {
            toast.error("Please enter a valid email address.");
            return;
        }
        if (!isPasswordValid(password())) {
            toast.error("Password must be between 3 and 64 characters.");
            return;
        }
        if (password() !== confirmPassword()) {
            toast.error("Passwords do not match.");
            return;
        }

        const currentSocket = getSocket(); // Get the current socket instance

        if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not connected. Please try again later.");
            return;
        }

        let messageHandler: ((event: MessageEvent) => void) | null = null;
        let errorHandler: ((event: Event) => void) | null = null;
        let closeHandler: (() => void) | null = null;

        const cleanupListeners = () => {
            if (messageHandler && currentSocket) currentSocket.removeEventListener('message', messageHandler);
            if (errorHandler && currentSocket) currentSocket.removeEventListener('error', errorHandler);
            if (closeHandler && currentSocket) currentSocket.removeEventListener('close', closeHandler);
            messageHandler = null;
            errorHandler = null;
            closeHandler = null;
        };
        
        messageHandler = (event: MessageEvent) => {
            try {
                const response = JSON.parse(event.data);
                if (response.type === "register_response") {
                    if (response.data.token) {
                        toast.success("Registration successful!");
                        localStorage.setItem("email", email());
                        localStorage.setItem("password", password());
                        localStorage.setItem("display_name", displayName());
                        props.onRegisterSuccess(); // Call the callback on successful registration
                    } else {
                        toast.error(
                            response.data.error
                                ? response.data.error.charAt(0).toUpperCase() + response.data.error.slice(1)
                                : "Registration failed. Please try again."
                        );
                    }
                    cleanupListeners();
                }
            } catch (err) {
                console.error("Failed to parse register response:", err);
                toast.error("Received an invalid response from server.");
                cleanupListeners();
            }
        };

        errorHandler = (error: Event) => {
            console.error("WebSocket error during registration:", error);
            toast.error("An error occurred during registration. Please try again later.");
            cleanupListeners();
        };

        closeHandler = () => {
            console.warn("WebSocket connection closed during registration attempt.");
            toast.error("Registration failed: Connection lost. Please try again.");
            cleanupListeners();
        };

        currentSocket.addEventListener('message', messageHandler);
        currentSocket.addEventListener('error', errorHandler);
        currentSocket.addEventListener('close', closeHandler);
        
        const salt = bcrypt.genSaltSync(10);
        const hashedPassword = bcrypt.hashSync(password(), salt);

        currentSocket.send(
            JSON.stringify({
                type: "register",
                data: {
                    display_name: displayName(),
                    email: email(),
                    hashed_password: hashedPassword,
                },
            })
        );
    };
    return (
        <div class="flex flex-col items-center p-6 border-2 border-gray-500 rounded-lg min-w-[350px] bg-gray-900 shadow-lg overflow-hidden">
            <p class="font-bold text-[3.5vh] mb-6">Register</p>
            <div class="w-full mb-4">
                <p class="text-[1.8vh] mb-1">Display Name:</p>
                <input
                    type="text"
                    class="w-full p-3 rounded bg-gray-800 text-[1.5vh] placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                    placeholder="John Doe"
                    value={displayName()}
                    onInput={e => setDisplayName(e.currentTarget.value)}
                />
            </div>
            <div class="w-full mb-4">
                <p class="text-[1.8vh] mb-1">Email ID:</p>
                <input
                    type="email"
                    class="w-full p-3 rounded bg-gray-800 text-[1.5vh] placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                    placeholder="john.doe@example.com"
                    value={email()}
                    onInput={e => setEmail(e.currentTarget.value)}
                />
            </div>
            <div class="w-full mb-4">
                <p class="text-[1.8vh] mb-1">Password:</p>
                <input
                    type="password"
                    class="w-full p-3 rounded bg-gray-800 text-[1.5vh] placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                    placeholder="Password"
                    value={password()}
                    onInput={e => setPassword(e.currentTarget.value)}
                />
            </div>
            <div class="w-full mb-6">
                <p class="text-[1.8vh] mb-1">Confirm Password:</p>
                <input
                    type="password"
                    class="w-full p-3 rounded bg-gray-800 text-[1.5vh] placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                    placeholder="Confirm Password"
                    value={confirmPassword()}
                    onInput={e => setConfirmPassword(e.currentTarget.value)}
                />
            </div>
            <button 
                class={`w-full py-3 mb-4 bg-green-500 text-white rounded-lg hover:bg-green-600 ${isFormValid()?'':'cursor-not-allowed opacity-50'}`}
                onClick={handleRegister}
                disabled={status() === "connecting" || status() === "reconnecting"}
            >
                Register
            </button>
            <div class="w-full text-center">
                <span class="text-gray-400 text-[1.5vh]">
                    Already have an account?{" "}
                    <a
                        href="#"
                        class="text-blue-400 hover:underline"
                        onClick={e => {
                            e.preventDefault();
                            props.onLoginClick();
                        }}
                    >
                        Login
                    </a>
                </span>
            </div>
        </div>
    );
};

const LoginScreen: Component<{ onLoginSuccess: () => void }> = (props) => {
    const [signUp, setSignUp] = createSignal<boolean>(false);
    return (
        <DesktopTemplate CurrentPage="Account">
            <div class="flex justify-center items-center w-full h-full pl-[2vh] pr-[1vh] py-[1vh] text-white">
                {signUp() ? (
                    <RegisterCard onLoginClick={() => setSignUp(false)} onRegisterSuccess={props.onLoginSuccess} />
                ) : (
                    <LoginCard onSignUpClick={() => setSignUp(true)} onLoginSuccess={props.onLoginSuccess} />
                )}
            </div>
        </DesktopTemplate>
    );
};

const AccountDetails: Component<{email: Accessor<string>; displayName: Accessor<string>}> = (props) => {
    const [open, setOpen] = createSignal(false);
    return (
        <Dialog open={open()} onOpenChange={setOpen}>
            <div class="bg-gray-950 border border-gray-700 rounded-md w-full p-[2vh] text-white flex flex-col shadow-md">
                <p class="font-semibold text-[2.5vh] mb-[2vh]">Account Details</p>
                <div class="mb-[1vh]">
                    <p class="text-gray-500 text-[1.5vh] uppercase tracking-wider">Display Name:</p>
                    <p class="text-[2vh]">{props.displayName()}</p>
                </div>
                <div class="mb-[1vh]">
                    <p class="text-gray-500 text-[1.5vh] uppercase tracking-wider">Email:</p>
                    <p class="text-[2vh] truncate">{props.email()}</p>
                </div>
                <div class="mb-[2vh]">
                    <p class="text-gray-500 text-[1.5vh] uppercase tracking-wider">Password:</p>
                    <p class="text-[2vh]">************</p>
                </div>
                <Dialog.Trigger class="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-[1vh] rounded mt-auto transition-colors duration-200 text-[1.5vh]">
                    Edit Account
                </Dialog.Trigger>
            </div>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 bg-black/50" />
                <Dialog.Content class="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-gray-800 p-6 rounded-md shadow-lg text-white w-[clamp(300px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-4">Edit Account Details</Dialog.Label>
                    {/* TODO: Complete this */}
                    <p class="mb-4">This is where the account editing form will go.</p>
                    <div class="flex justify-end space-x-2">
                        <Dialog.Close class="bg-gray-600 hover:bg-gray-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Cancel
                        </Dialog.Close>
                        <button class="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Save Changes
                        </button>
                    </div>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    )
}

const DangerZone: Component<{logout: () => void; class?: string}> = (props) => {
    return (
        <div class={`bg-red-950 border border-red-700 rounded-md p-[2vh] text-white flex flex-col shadow-md w-full ${props.class}`}>
            <p class="font-semibold text-[2vh] mb-[2vh] text-center">Danger Zone</p>
            <div class="w-full flex space-x-[1vh]">
                <button class="bg-red-600 hover:bg-red-700 w-full text-white font-semibold py-[1vh] px-[1vw] rounded mt-auto transition-colors duration-200 text-[1.5vh]">
                    Delete&nbsp;Account
                </button>
                <button class="bg-yellow-600 w-full hover:bg-yellow-700 text-white font-semibold py-[1vh] px-[1vw] rounded mt-auto transition-colors duration-200 text-[1.5vh]"
                onClick={() => {
                    props.logout();
                }}
                >
                    Log Out
                </button>
            </div>
        </div>
    )
}

const UserStat: Component<{title: string; value: string; class?: string}> = (props) => {
    return (
        <div class={`bg-neutral-900 w-full h-full rounded-md border border-neutral-700 flex justify-between items-center flex-col text-white p-[1vh] ${props.class ?? ''}`}>
            <p class="text-blue-700 font-bold text-[2.5vh]">{props.title}</p>
            <p class="text-[2vh]">{props.value}</p>
        </div>
    )
}

const AccountManager: Component<{logout: () => void}> = (props) => {

    const [email] = createSignal(localStorage.getItem("email") || "{email}");
    const [displayName] = createSignal(localStorage.getItem("display_name") || "{display_name}");
    const [spaceUsed] = createSignal("10 GB");
    const [filesHosted] = createSignal("100");
    const [collections] = createSignal("5");

    return (
        <DesktopTemplate CurrentPage="Account">
            <div class="w-full h-full flex justify-center items-center space-x-[1vh]">
                <div class="flex flex-col w-[20%] space-y-[2vh]">
                    <AccountDetails email={email} displayName={displayName}/>
                </div>
                <div class="min-w-[20%] grid grid-cols-2 grid-rows-2 gap-[1vh]">
                    <UserStat title="Space&nbsp;Used" value={spaceUsed()} class="col-span-2"/>
                    <UserStat title="Files&nbsp;Hosted" value={filesHosted()}/>
                    <UserStat title="Collections" value={collections()}/>
                    <DangerZone logout={props.logout} class="col-span-2"/>
                </div>
            </div>
        </DesktopTemplate>
    )
}

const Account: Component = () => {
    const [isLoggedIn, setIsLoggedIn] = createSignal(false);

    const handleLoginSuccess = () => {
        setIsLoggedIn(true);
    };

    const handleLogout = () => {
        localStorage.removeItem("email");
        localStorage.removeItem("password");
        localStorage.removeItem("display_name");
        setIsLoggedIn(false);
        toast('Logged out successfully!', {
            icon: '↩️'
        })
    };

    onMount(() => {
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");
        if (storedEmail && storedPassword) {
            setIsLoggedIn(true);
        } else {
            setIsLoggedIn(false);
        }
    });
    
    return (
        <>
            <title>Account | DriveV3</title>
            {isLoggedIn() ? <AccountManager logout={handleLogout}/> : <LoginScreen onLoginSuccess={handleLoginSuccess} />}
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