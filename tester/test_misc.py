"""Miscellaneous tests that don't fit neatly into the other modules.

  * unknown message type (server should ignore it and send nothing)
  * import_from_github with an invalid URL (should error immediately)
"""

from .harness import check, check_in
from .helpers import make_msg, open_ws, recv_until, send_and_wait
from .test_accounts import register_and_login


async def test_unknown_message_type():
    """Send a message type the server doesn't recognise.

    The server logs "Unknown message type" and sends nothing back, so we expect
    no response within a short window.
    """
    print("\n[test] unknown message type")
    ws = await open_ws()
    await ws.send(make_msg("definitely_not_a_real_type", {}))
    got = await recv_until(ws, lambda m: True, timeout=1.5)
    check("no response for unknown type", got is None)
    await ws.close()


async def test_import_from_github_invalid():
    """Import a GitHub repo with an invalid URL; expect an immediate error.

    Error responses use type ``"error"``, so we wait for that rather than the
    success notification type.
    """
    print("\n[test] import from github (invalid url)")
    email, password = await register_and_login()
    ws = await open_ws()

    resp = await send_and_wait(ws, "import_from_github",
                               {"repo_url": "not-a-url",
                                "auth": {"email": email, "password": password}},
                               "error")
    check("invalid github url errors", resp is not None)
    if resp:
        check_in("invalid url error msg", "invalid github url", resp["data"])

    await ws.close()