import { Accessor, Component, createEffect, createSignal, onMount, useContext } from "solid-js";
import Dialog from '@corvu/dialog';
import { toast, Toaster } from "solid-toast";
import { useWebSocket } from "../Websockets";
import bcrypt from "bcryptjs";
import { DesktopTemplate } from "../components/Template";
import { AppContext } from "../Context";
import { fetchFilesAndCollections, formatFileSize, handleLogout } from "../library/functions";

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
                        localStorage.removeItem("token");
                        props.onLoginSuccess(); // Call the callback on successful login
                        // TODO: Setup user migration
                        // for now, just remove the previous token
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
                        localStorage.removeItem("token");
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

const AccountDetails: Component<{email: Accessor<string>; setEmail: (email: string) => void; displayName: Accessor<string>; setDisplayName: (name: string) => void}> = (props) => {
    const [open, setOpen] = createSignal(false);
    const [tempDisplayName, setTempDisplayName] = createSignal(props.displayName());
    const [tempEmail, setTempEmail] = createSignal(props.email());
    const [tempNewPassword, setTempNewPassword] = createSignal("");
    const [currentAuthPassword, setCurrentAuthPassword] = createSignal("");
    const [hasPendingChanges, setHasPendingChanges] = createSignal(false);

    const { socket: getSocket, status: socketStatus } = useWebSocket();

    createEffect(() => {
        const dnChanged = tempDisplayName() !== props.displayName();
        const emailChanged = tempEmail() !== props.email();
        const pwChanged = tempNewPassword() !== "";
        setHasPendingChanges(dnChanged || emailChanged || pwChanged);
    });

    const resetDialogFormState = () => {
        setTempDisplayName(props.displayName());
        setTempEmail(props.email());
        setTempNewPassword("");
        setCurrentAuthPassword("");
        // hasPendingChanges will be updated by the createEffect
    };

    const handleSaveChanges = async () => {
        const currentSocket = getSocket();

        if (socketStatus() === "connecting" || socketStatus() === "reconnecting") {
            toast.error("WebSocket is connecting. Please wait and try again.");
            return;
        }

        if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not connected. Please try again later.");
            return;
        }

        if (!hasPendingChanges()) {
            toast.error("No changes to save.");
            setOpen(false);
            return;
        }

        if (!currentAuthPassword()) {
            toast.error("Please enter your current password to authenticate changes.");
            return;
        }

        const initialEmailForAuth = props.email(); // Email at the time of opening dialog
        const authPasswordForRequests = currentAuthPassword();

        let anyChangeSucceeded = false;
        let allOperationsAttemptedAndSuccessful = true; // Assume success until a failure or skipped operation

        const sendUpdateRequest = (
            requestType: string,
            payload: any,
            successMessage: string,
            successCallback?: (data: any) => void
        ): Promise<boolean> => {
            return new Promise<boolean>((resolve) => {
                if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN) {
                    toast.error(`WebSocket connection lost before sending ${requestType}.`);
                    resolve(false);
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
                        if (response.type === `${requestType}_response`) {
                            cleanupListeners();
                            if (response.data.error) {
                                toast.error(
                                    response.data.error === "record not found" || response.data.error === "invalid credentials"
                                        ? "Authentication failed. Check current password."
                                        : response.data.error.charAt(0).toUpperCase() + response.data.error.slice(1)
                                );
                                resolve(false);
                            } else {
                                toast.success(successMessage);
                                if (successCallback) successCallback(response.data);
                                anyChangeSucceeded = true;
                                resolve(true);
                            }
                        }
                    } catch (err) {
                        cleanupListeners();
                        console.error(`Failed to parse ${requestType} response:`, err);
                        toast.error("Received an invalid response from server.");
                        resolve(false);
                    }
                };

                errorHandler = (error: Event) => {
                    cleanupListeners();
                    console.error(`WebSocket error during ${requestType}:`, error);
                    toast.error(`An error occurred while updating ${requestType.replace("_", " ")}.`);
                    resolve(false);
                };

                closeHandler = () => {
                    if (messageHandler) { // Only if not already cleaned up
                        cleanupListeners();
                        console.warn(`WebSocket connection closed during ${requestType} attempt.`);
                        toast.error(`Update for ${requestType.replace("_", " ")} failed: Connection lost.`);
                        resolve(false);
                    }
                };

                currentSocket.addEventListener('message', messageHandler);
                currentSocket.addEventListener('error', errorHandler);
                currentSocket.addEventListener('close', closeHandler);

                currentSocket.send(JSON.stringify({ type: requestType, data: payload }));
            });
        };

        // --- Change Display Name ---
        if (tempDisplayName() !== props.displayName()) {
            if (tempDisplayName().length < 3 || tempDisplayName().length > 64) {
                toast.error("Display name must be between 3 and 64 characters.");
                allOperationsAttemptedAndSuccessful = false;
            } else {
                const success = await sendUpdateRequest(
                    "change_display_name",
                    {
                        email: initialEmailForAuth,
                        password: authPasswordForRequests,
                        display_name: tempDisplayName(),
                    },
                    "Display name updated!",
                    () => {
                        localStorage.setItem("display_name", tempDisplayName());
                        props.setDisplayName(tempDisplayName());
                    }
                );
                if (!success) allOperationsAttemptedAndSuccessful = false;
            }
        }

        // --- Change Email ---
        if (tempEmail() !== props.email() && allOperationsAttemptedAndSuccessful) { // Proceed if previous steps were fine
            if (!isEmailValid(tempEmail())) {
                toast.error("Please enter a valid new email address.");
                allOperationsAttemptedAndSuccessful = false;
            } else {
                const success = await sendUpdateRequest(
                    "change_email",
                    {
                        old_email: initialEmailForAuth,
                        password: authPasswordForRequests,
                        new_email: tempEmail(),
                    },
                    "Email updated!",
                    () => {
                        localStorage.setItem("email", tempEmail());
                        props.setEmail(tempEmail());
                    }
                );
                if (!success) allOperationsAttemptedAndSuccessful = false;
            }
        }

        // --- Change Password ---
        if (tempNewPassword() && allOperationsAttemptedAndSuccessful) { // Proceed if previous steps were fine
            if (!isPasswordValid(tempNewPassword())) {
                toast.error("New password must be between 3 and 64 characters.");
                allOperationsAttemptedAndSuccessful = false;
            } else {
                const salt = bcrypt.genSaltSync(10);
                const hashedPassword = bcrypt.hashSync(tempNewPassword(), salt);
                const success = await sendUpdateRequest(
                    "change_password",
                    {
                        email: localStorage.getItem("email") || initialEmailForAuth, // Use potentially updated email
                        old_password: authPasswordForRequests,
                        new_password_hashed: hashedPassword,
                    },
                    "Password updated!",
                    () => localStorage.setItem("password", tempNewPassword())
                );
                if (!success) allOperationsAttemptedAndSuccessful = false;
            }
        }

        if (anyChangeSucceeded) {
            if (allOperationsAttemptedAndSuccessful) {
                setOpen(false); // Close dialog only if all attempted operations were successful
                // resetDialogFormState() is called by onOpenChange when dialog closes
            } else {
                toast.error("Some changes may not have been saved. Please review.");
                // Refresh temp states from localStorage to reflect partial successes
                setTempDisplayName(localStorage.getItem("display_name") || props.displayName());
                setTempEmail(localStorage.getItem("email") || props.email());
                // Don't clear tempNewPassword or currentAuthPassword if a password change was attempted and failed
            }
        } else if (hasPendingChanges()) { // No change succeeded but changes were attempted
            toast.error("Failed to save changes. Please check your current password and details.");
        }
    };

    return (
        <Dialog open={open()} onOpenChange={(newOpen) => {
            setOpen(newOpen);
            if (!newOpen) {
                resetDialogFormState();
            } else {
                // Ensure form is reset to current prop values when opening
                setTempDisplayName(props.displayName());
                setTempEmail(props.email());
                setTempNewPassword("");
                setCurrentAuthPassword("");
            }
        }}>
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
                <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
                <Dialog.Content class="flex z-50 justify-center flex-col fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-neutral-800 p-6 rounded-md shadow-lg text-white w-[clamp(300px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-2 text-center">Edit Account Details</Dialog.Label>
                    <p class="mb-1 text-sm text-neutral-400">Change your details below.</p>
                    <p class="mb-4 text-sm text-neutral-400">Enter your <strong class="text-neutral-300">current password</strong> to save any changes.</p>
                    
                    <label for="displayNameInput" class="text-sm text-neutral-300 mb-1 mt-2">Display Name:</label>
                    <input
                        id="displayNameInput"
                        type="text"
                        class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                        placeholder="New Display Name"
                        value={tempDisplayName()}
                        onInput={e => setTempDisplayName(e.currentTarget.value)}
                        autocomplete="off"
                    />
                    <label for="emailInput" class="text-sm text-neutral-300 mb-1 mt-3">Email ID:</label>
                    <input
                        id="emailInput"
                        type="email"
                        class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                        placeholder="New Email ID"
                        value={tempEmail()}
                        onInput={e => setTempEmail(e.currentTarget.value)}
                        autocomplete="off"
                    />
                    <label for="newPasswordInput" class="text-sm text-neutral-300 mb-1 mt-3">New Password (optional):</label>
                    <input
                        id="newPasswordInput"
                        type="password"
                        class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                        placeholder="Leave blank to keep current password"
                        value={tempNewPassword()}
                        onInput={e => setTempNewPassword(e.currentTarget.value)}
                        autocomplete="new-password"
                    />
                    
                    <div class={`${hasPendingChanges() ? 'mt-3' : 'hidden'}`}>
                        <label for="currentPasswordInput" class="text-sm text-red-400 mb-1 font-semibold">Current Password (required to save):</label>
                        <input
                            id="currentPasswordInput"
                            type="password"
                            class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-red-400"
                            placeholder="Enter Current Password"
                            value={currentAuthPassword()}
                            onInput={e => setCurrentAuthPassword(e.currentTarget.value)}
                            autocomplete="current-password"
                        />
                    </div>
                    
                    <div class="flex justify-end space-x-3 mt-6">
                        <Dialog.Close class="bg-neutral-600 hover:bg-neutral-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Cancel
                        </Dialog.Close>
                        <button
                            class={`bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200 ${(!hasPendingChanges() || !currentAuthPassword() || socketStatus() !== 'connected') ? 'opacity-50 cursor-not-allowed' : ''}`}
                            onClick={handleSaveChanges}
                            disabled={!hasPendingChanges() || !currentAuthPassword() || socketStatus() !== 'connected'}
                        >
                            Save Changes
                        </button>
                    </div>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    )
}

const DeleteAccountDialog: Component = () => {
    const [open, setOpen] = createSignal(false);
    const [deletePassword, setDeletePassword] = createSignal("");
    const { socket: getSocket, status: socketStatus } = useWebSocket();
    const handleAccountDeletion = () => {
        if (socketStatus() !== "connected") {
            setTimeout(() => {
                handleAccountDeletion();
            }, 100);
            return;
        }
        if (!deletePassword()) {
            toast.error("Please enter your password to confirm account deletion.");
            return;
        }
        const currentSocket = getSocket();
        currentSocket?.send(
            JSON.stringify({
                type: "delete_account",
                data: {
                    email: localStorage.getItem("email"),
                    password: deletePassword(),
                },
            })
        )
    }
    return (
        <Dialog open={open()} onOpenChange={(newOpen) => {
            setOpen(newOpen);
            setDeletePassword("");
        }}>
            <Dialog.Trigger class="bg-red-600 w-full hover:bg-red-700 text-white font-semibold py-[1vh] px-[1vw] rounded mt-auto transition-colors duration-200 text-[1.5vh]">
                Delete&nbsp;Account
            </Dialog.Trigger>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
                <Dialog.Content class="flex z-50 justify-center flex-col fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-neutral-800 p-6 rounded-md shadow-lg text-white w-[clamp(300px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-2 text-center">Confirm Account Deletion</Dialog.Label>
                    <p class="mb-4 text-sm text-neutral-400">
                        P.S. if u delete ur account then i cant recover it even if you ask me to, all ur files, collections, and data will be instantly deleted automatically
                    </p>
                    <label for="deletePasswordInput" class="text-sm text-red-400 mb-1 font-semibold">
                        Enter ur password to confirm:
                    </label>
                    <input
                        id="deletePasswordInput"
                        type="password"
                        class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-red-400"
                        placeholder="Your Password"
                        autocomplete="current-password"
                        onInput={e => setDeletePassword(e.currentTarget.value)}
                    />
                    <div class="flex justify-end space-x-3 mt-6">
                        <Dialog.Close class="bg-neutral-600 hover:bg-neutral-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Cancel
                        </Dialog.Close>
                        <button
                            class="bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200"
                            onClick={handleAccountDeletion}
                        >
                            Delete Account
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
                <DeleteAccountDialog/>
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

    const [email, setEmail] = createSignal(localStorage.getItem("email") || "{email}");
    const [displayName, setDisplayName] = createSignal(localStorage.getItem("display_name") || "{display_name}");
    const ctx = useContext(AppContext)!;

    return (
        <DesktopTemplate CurrentPage="Account">
            <div class="w-full h-full flex justify-center items-center space-x-[1vh]">
                <div class="flex flex-col min-w-[20%] space-y-[2vh]">
                    <AccountDetails email={email} setEmail={setEmail} displayName={displayName} setDisplayName={setDisplayName}/>
                </div>
                <div class="min-w-[20%] grid grid-cols-2 grid-rows-2 gap-[1vh]">
                    <UserStat title="Space&nbsp;Used" value={formatFileSize(ctx.files().reduce((sum, file) => sum + file.file_size, 0))} class="col-span-2"/>
                    <UserStat title="Files&nbsp;Hosted" value={ctx.files().length.toString()} />
                    <UserStat title="Collections" value={ctx.userCollections().size.toString()}/>
                    <DangerZone logout={props.logout} class="col-span-2"/>
                </div>
            </div>
        </DesktopTemplate>
    )
}

const Account: Component = () => {
    const [isLoggedIn, setIsLoggedIn] = createSignal(false);
    const ctx = useContext(AppContext)!;

    const currentSocket = useWebSocket();

    const handleLoginSuccess = () => {
        setIsLoggedIn(true);
        fetchFilesAndCollections(currentSocket.socket()!);
    };

    onMount(() => {
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");
        setIsLoggedIn(!!(storedEmail && storedPassword));
    });

    window.addEventListener('storage', () => {
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");
        setIsLoggedIn(!!(storedEmail && storedPassword));
    });
    
    return (
        <>
            <title>Account | DriveV3</title>
            {isLoggedIn() ? <AccountManager logout={() => handleLogout(setIsLoggedIn, ctx)} /> : <LoginScreen onLoginSuccess={handleLoginSuccess} />}
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