import { Component, createSignal } from "solid-js";
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

const LoginCard: Component<{ onSignUpClick: () => void }> = (props) => {
    const [email, setEmail] = createSignal("");
    const [password, setPassword] = createSignal("");
    const { socket, status } = useWebSocket();

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
        if (status() !== "connected") {
            toast.error("WebSocket is not connected. Please try again later.");
            return;
        }
        socket.send(
            JSON.stringify({
                type: "login",
                data: {
                    email: email(),
                    password: password(),
                },
            })
        );
        socket.onmessage = (event) => {
            const response = JSON.parse(event.data);
            if (response.type === "login_response") {
                if (response.data.token) {
                    toast.success("Login successful!");
                    localStorage.setItem("email", email());
                    localStorage.setItem("password", password());
                } else {
                    toast.error(
                        response.data.error
                            ? response.data.error.charAt(0).toUpperCase() + response.data.error.slice(1)
                            : "Login failed. Please try again."
                    );
                }
            }
        }
        socket.onerror = (error) => {
            console.error("WebSocket error:", error);
            toast.error("An error occurred while connecting to the server. Please try again later.");
        }
        socket.onclose = () => {
            console.warn("WebSocket connection closed.");
            toast.error("WebSocket connection closed. Please refresh the page to try again.");
        }
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

const RegisterCard: Component<{ onLoginClick: () => void }> = (props) => {
    const [displayName, setDisplayName] = createSignal("");
    const [email, setEmail] = createSignal("");
    const [password, setPassword] = createSignal("");
    const [confirmPassword, setConfirmPassword] = createSignal("");
    const { socket, status } = useWebSocket();
    
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
        if (status() !== "connected") {
            toast.error("WebSocket is not connected. Please try again later.");
            return;
        }
        const salt = bcrypt.genSaltSync(10);
        const hashedPassword = bcrypt.hashSync(password(), salt);

        socket.send(
            JSON.stringify({
                type: "register",
                data: {
                    display_name: displayName(),
                    email: email(),
                    hashed_password: hashedPassword,
                },
            })
        );
        socket.onmessage = (event) => {
            const response = JSON.parse(event.data);
            if (response.type === "register_response") {
                if (response.data.token) {
                    toast.success("Registration successful!");
                    localStorage.setItem("email", email());
                    localStorage.setItem("password", password());
                } else {
                    toast.error(
                        response.data.error
                            ? response.data.error.charAt(0).toUpperCase() + response.data.error.slice(1)
                            : "Registration failed. Please try again."
                    );
                }
            }
        }
        socket.onerror = (error) => {
            console.error("WebSocket error:", error);
            toast.error("An error occurred while connecting to the server. Please try again later.");
        }
        socket.onclose = () => {
            console.warn("WebSocket connection closed.");
            toast.error("WebSocket connection closed. Please refresh the page to try again.");
        }
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

const AccountManager: Component = () => {
    const [signUp, setSignUp] = createSignal<boolean>(false);
    return (
        <DesktopTemplate CurrentPage="Account">
            <div class="flex justify-center items-center w-full h-full pl-[2vh] pr-[1vh] py-[1vh] text-white">
                {signUp() ? (
                    <RegisterCard onLoginClick={() => setSignUp(false)} />
                ) : (
                    <LoginCard onSignUpClick={() => setSignUp(true)} />
                )}
            </div>
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
        </DesktopTemplate>
    );
};
export default AccountManager;