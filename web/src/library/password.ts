import bcrypt from "bcryptjs";

/**
 * Password handling for the SHA-256 pre-hash scheme.
 *
 * The plaintext password never leaves the browser. Instead:
 *   - registration sends bcrypt(sha256(password)) to the backend,
 *   - login and every authenticated request send sha256(password),
 *   - the backend stores bcrypt(sha256(password)).
 *
 * The digest (not the plaintext) is cached in localStorage, so existing code
 * that forwards `localStorage.getItem("password")` keeps working unchanged.
 */

/** localStorage key marking that `password` holds a SHA-256 digest. */
const PASSWORD_SCHEME_KEY = "password_scheme";
const PASSWORD_SCHEME_SHA256 = "sha256";

/** Return the lowercase hex SHA-256 digest of `password`. */
async function sha256Hex(password: string): Promise<string> {
    const data = new TextEncoder().encode(password);
    const digest = await crypto.subtle.digest("SHA-256", data);
    return Array.from(new Uint8Array(digest))
        .map(b => b.toString(16).padStart(2, "0"))
        .join("");
}

/**
 * Return bcrypt(sha256(password)) for storage on the backend.
 *
 * The digest is 64 ASCII characters, comfortably under bcrypt's 72-byte input
 * limit. Do not switch to a longer encoding (e.g. base64 of the raw digest)
 * without revisiting that limit.
 */
async function hashPasswordForStorage(password: string): Promise<string> {
    const digest = await sha256Hex(password);
    const salt = bcrypt.genSaltSync(10);
    return bcrypt.hashSync(digest, salt);
}

/** True when the cached password is already a SHA-256 digest. */
function isPasswordMigrated(): boolean {
    return localStorage.getItem(PASSWORD_SCHEME_KEY) === PASSWORD_SCHEME_SHA256;
}

/** Cache the digest (never the plaintext) and mark the scheme. */
function storePasswordDigest(digest: string): void {
    localStorage.setItem("password", digest);
    localStorage.setItem(PASSWORD_SCHEME_KEY, PASSWORD_SCHEME_SHA256);
}

/** Remove all cached credentials, including the scheme marker. */
function clearStoredCredentials(): void {
    localStorage.removeItem("email");
    localStorage.removeItem("password");
    localStorage.removeItem("display_name");
    localStorage.removeItem(PASSWORD_SCHEME_KEY);
}

interface WsResult {
    ok: boolean;
    data: any;
}

/**
 * Send a request over the websocket and resolve with the first response of
 * `responseType`. Error payloads (a string, or an object with `error`) resolve
 * with `ok: false`.
 */
function wsRequest(
    ws: WebSocket,
    type: string,
    data: unknown,
    responseType: string,
    timeoutMs = 15000,
): Promise<WsResult> {
    return new Promise<WsResult>((resolve) => {
        let settled = false;
        const finish = (result: WsResult) => {
            if (settled) return;
            settled = true;
            clearTimeout(timer);
            ws.removeEventListener("message", onMessage);
            ws.removeEventListener("close", onClose);
            resolve(result);
        };
        const onMessage = (event: MessageEvent) => {
            let message: any;
            try {
                message = JSON.parse(event.data);
            } catch {
                return;
            }
            if (message.type !== responseType) return;
            const payload = message.data;
            const isError =
                typeof payload === "string" ||
                (payload && typeof payload === "object" && "error" in payload);
            finish({ ok: !isError, data: payload });
        };
        const onClose = () => finish({ ok: false, data: "connection closed" });
        const timer = setTimeout(() => finish({ ok: false, data: "timeout" }), timeoutMs);

        ws.addEventListener("message", onMessage);
        ws.addEventListener("close", onClose);
        ws.send(JSON.stringify({ type, data }));
    });
}

/**
 * Log in with a plaintext password.
 *
 * Sends the SHA-256 digest. If the account has not been migrated yet the
 * backend replies with `legacy_password`; we then retry once with the plaintext
 * and `legacy: true`, which upgrades the account server-side.
 *
 * On success the digest is cached in localStorage.
 */
async function loginWithPassword(
    ws: WebSocket,
    email: string,
    plaintext: string,
): Promise<WsResult> {
    const digest = await sha256Hex(plaintext);
    let result = await wsRequest(ws, "login", { email, password: digest }, "login_response");

    if (!result.ok && result.data === "legacy_password") {
        result = await wsRequest(
            ws,
            "login",
            { email, password: plaintext, legacy: true },
            "login_response",
        );
    }

    if (result.ok) {
        storePasswordDigest(digest);
    }
    return result;
}

/**
 * Migrate a plaintext password cached by the pre-migration frontend.
 *
 * Returns true when the cached credentials are usable afterwards (either the
 * migration succeeded or the account was already migrated). Returns false when
 * the cached plaintext is stale, in which case the caller should clear the
 * credentials.
 */
async function migrateCachedPassword(ws: WebSocket, email: string, plaintext: string): Promise<boolean> {
    const digest = await sha256Hex(plaintext);
    const newHash = await hashPasswordForStorage(plaintext);
    const result = await wsRequest(
        ws,
        "migrate_password",
        { email, old_password: plaintext, new_password_hashed: newHash },
        "migrate_password_response",
    );
    if (!result.ok) return false;
    storePasswordDigest(digest);
    return true;
}

export {
    PASSWORD_SCHEME_KEY,
    sha256Hex,
    hashPasswordForStorage,
    isPasswordMigrated,
    storePasswordDigest,
    clearStoredCredentials,
    wsRequest,
    loginWithPassword,
    migrateCachedPassword,
};
export type { WsResult };
