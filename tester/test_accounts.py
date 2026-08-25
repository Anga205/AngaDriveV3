"""Account lifecycle tests.

Covers the account-related websocket message types:
  * register
  * login
  * change_display_name
  * change_email
  * change_password
  * delete_account

Each test creates a fresh account (via :func:`register_and_login`) so tests are
independent and can run in any order. Several tests need multiple websocket
connections to verify cross-connection behaviour (e.g. ``force_logout`` after
account deletion reaches a *different* connection than the one that deleted the
account).
"""

from . import config
from .harness import check, check_eq, check_in
from .helpers import (bcrypt_hash, open_ws, random_email, recv_until,
                      send_and_wait)


async def register_and_login():
    """Register a fresh account and return ``(email, password)``.

    This is a setup helper used by many tests. It also verifies the basic
    register + login round-trip as a side effect.
    """
    email = random_email()
    password = "secret123"
    hashed = bcrypt_hash(password)

    ws = await open_ws()
    resp = await send_and_wait(ws, "register",
                               {"display_name": "Tester", "email": email,
                                "hashed_password": hashed},
                               "register_response")
    check("register returns account", resp is not None)
    if resp:
        check_eq("register has email", resp["data"].get("email"), email)
        check("register returns token", bool(resp["data"].get("token")))

    # Registering the same email again must fail.
    resp2 = await send_and_wait(ws, "register",
                                {"display_name": "Tester2", "email": email,
                                 "hashed_password": hashed},
                                "error")
    check("duplicate register errors", resp2 is not None)
    if resp2:
        check_in("duplicate register error msg", "already exists", resp2["data"])

    # Login with the plaintext password.
    resp3 = await send_and_wait(ws, "login",
                                {"email": email, "password": password},
                                "login_response")
    check("login returns account", resp3 is not None)
    if resp3:
        check_eq("login email", resp3["data"].get("email"), email)
        check("login returns token", bool(resp3["data"].get("token")))

    # Login with a wrong password must fail (type stays login_response).
    resp4 = await send_and_wait(ws, "login",
                                {"email": email, "password": "wrongpass"},
                                "login_response")
    check("bad login errors", resp4 is not None)
    if resp4:
        check_in("bad login error msg", "invalid credentials", resp4["data"])

    await ws.close()
    return email, password


async def test_change_display_name():
    """Change a user's display name and verify the response reflects it."""
    print("\n[test] change display name")
    email, password = await register_and_login()
    ws = await open_ws()
    resp = await send_and_wait(ws, "change_display_name",
                               {"display_name": "NewName", "email": email,
                                "password": password},
                               "change_display_name_response")
    check("change display name returns account", resp is not None)
    if resp:
        check_eq("display name updated", resp["data"].get("display_name"), "NewName")
    await ws.close()


async def test_change_email():
    """Change a user's email and verify the response reflects it."""
    print("\n[test] change email")
    email, password = await register_and_login()
    new_email = random_email()
    ws = await open_ws()
    resp = await send_and_wait(ws, "change_email",
                               {"old_email": email, "new_email": new_email,
                                "password": password},
                               "change_email_response")
    check("change email returns account", resp is not None)
    if resp:
        check_eq("email updated", resp["data"].get("email"), new_email)
    await ws.close()


async def test_change_password():
    """Change a password, then verify the old one is rejected and the new one works.

    This requires a second websocket connection to log in with the new password
    after the change.
    """
    print("\n[test] change password")
    email, password = await register_and_login()
    new_password = "newsecret456"
    ws = await open_ws()
    resp = await send_and_wait(ws, "change_password",
                               {"email": email, "old_password": password,
                                "new_password_hashed": bcrypt_hash(new_password)},
                               "change_password_response")
    check("change password returns account", resp is not None)
    if resp:
        check_eq("password change email", resp["data"].get("email"), email)

    # Old password should no longer work; new one should.
    ws2 = await open_ws()
    old_login = await send_and_wait(ws2, "login",
                                    {"email": email, "password": password},
                                    "login_response")
    check("old password rejected", old_login is not None and "invalid" in old_login["data"])
    new_login = await send_and_wait(ws2, "login",
                                    {"email": email, "password": new_password},
                                    "login_response")
    check("new password accepted", new_login is not None and "token" in new_login["data"])
    await ws.close()
    await ws2.close()


async def test_delete_account():
    """Delete an account and verify cross-connection force_logout + failed re-login.

    Sequence:
      1. Connection A authenticates as the user (so it is eligible for logout).
      2. Connection B deletes the account.
      3. Connection A should receive a ``force_logout`` message.
      4. A fresh login with the deleted credentials must fail.
    """
    print("\n[test] delete account")
    email, password = await register_and_login()

    # Connection A authenticates as the user (so it should get force_logout).
    ws_a = await open_ws()
    await send_and_wait(ws_a, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")

    # Connection B performs the deletion.
    ws_b = await open_ws()
    resp = await send_and_wait(ws_b, "delete_account",
                               {"email": email, "password": password},
                               "success_notification")
    check("delete account succeeds", resp is not None)
    if resp:
        check_in("delete account message", "deleted", resp["data"])

    # Connection A should be force-logged-out.
    logout = await recv_until(ws_a, lambda m: m.get("type") == "force_logout", timeout=config.DEFAULT_TIMEOUT)
    check("force_logout received on authed conn", logout is not None)
    if logout:
        check_eq("force_logout email", logout["data"], email)

    # Login should now fail.
    ws_c = await open_ws()
    login = await send_and_wait(ws_c, "login",
                                {"email": email, "password": password},
                                "login_response")
    check("login after delete fails", login is not None and "invalid" in login["data"])

    await ws_a.close()
    await ws_b.close()
    await ws_c.close()