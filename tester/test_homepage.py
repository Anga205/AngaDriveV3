"""Homepage pulse tests.

Covers the ``enable_homepage_updates`` message type and the homepage pulses
that are broadcast to subscribed connections when the site state changes.

When a connection enables homepage updates, the server immediately sends 5
messages:
  1. graph_data (site activity)
  2. graph_data (space used)
  3. user_count
  4. files_hosted_count
  5. system_information

After that, the connection receives periodic pulses (e.g. ``files_hosted_count``
when a file is uploaded). These tests verify both the immediate burst and the
subsequent pulse behaviour.
"""

import asyncio
import json

import websockets

from .harness import check
from .helpers import make_msg, open_ws, recv_until, send_and_wait, upload_file
from .test_accounts import register_and_login


async def test_enable_homepage_updates():
    """Enable homepage updates and verify the immediate 5-message burst."""
    print("\n[test] enable homepage updates")
    ws = await open_ws()
    await ws.send(make_msg("enable_homepage_updates", True))

    # Expect 5 immediate messages: 2x graph_data, user_count, files_hosted_count, system_information.
    types = []
    try:
        async with asyncio.timeout(5.0):
            while len(types) < 5:
                raw = await ws.recv()
                msg = json.loads(raw)
                types.append(msg.get("type"))
    except (asyncio.TimeoutError, websockets.ConnectionClosed):
        pass

    check("received 5 homepage messages", len(types) == 5, f"(got {types})")
    check("has graph_data", types.count("graph_data") >= 2)
    check("has user_count", "user_count" in types)
    check("has files_hosted_count", "files_hosted_count" in types)
    check("has system_information", "system_information" in types)
    await ws.close()


async def test_homepage_pulse_on_upload():
    """Verify a homepage subscriber receives a pulse when a file is uploaded.

    Sequence:
      1. Connection A subscribes to homepage updates and drains the initial burst.
      2. Connection B uploads a file.
      3. Connection A should receive a ``files_hosted_count`` pulse.
    """
    print("\n[test] homepage pulse reaches subscriber on upload")
    email, password = await register_and_login()

    # Connection A subscribes to homepage updates.
    ws_a = await open_ws()
    await ws_a.send(make_msg("enable_homepage_updates", True))
    # Drain the 5 immediate messages.
    try:
        async with asyncio.timeout(5.0):
            for _ in range(5):
                await ws_a.recv()
    except (asyncio.TimeoutError, websockets.ConnectionClosed):
        pass

    # Connection B uploads a file.
    ws_b = await open_ws()
    await send_and_wait(ws_b, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")
    await upload_file(email=email, password=password, filename="home.txt",
                      content=b"homepage pulse")

    # Connection A should receive a files_hosted_count pulse.
    pulse = await recv_until(ws_a, lambda m: m.get("type") == "files_hosted_count")
    check("files_hosted_count pulse received", pulse is not None)
    if pulse:
        check("count is an int", isinstance(pulse["data"], int))

    await ws_a.close()
    await ws_b.close()


async def test_homepage_pulse_reaches_all_subscribers():
    """Verify an upload updates ALL active homepage subscribers.

    Sequence:
      1. Connections A and B both subscribe to homepage updates and drain the
         initial burst.
      2. Connection C uploads a file.
      3. Both A and B should receive a ``files_hosted_count`` pulse.
    """
    print("\n[test] homepage pulse reaches all subscribers on upload")
    email, password = await register_and_login()

    # Two homepage subscribers.
    subscribers = []
    for _ in range(2):
        ws = await open_ws()
        await ws.send(make_msg("enable_homepage_updates", True))
        # Drain the 5 immediate messages.
        try:
            async with asyncio.timeout(5.0):
                for _ in range(5):
                    await ws.recv()
        except (asyncio.TimeoutError, websockets.ConnectionClosed):
            pass
        subscribers.append(ws)

    # Connection C uploads a file.
    ws_c = await open_ws()
    await send_and_wait(ws_c, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")
    await upload_file(email=email, password=password, filename="multi_home.txt",
                      content=b"multi homepage pulse")

    # Both subscribers should receive a files_hosted_count pulse.
    for i, ws in enumerate(subscribers):
        pulse = await recv_until(ws, lambda m: m.get("type") == "files_hosted_count")
        check(f"subscriber {i} received files_hosted_count", pulse is not None)
        if pulse:
            check(f"subscriber {i} count is int", isinstance(pulse["data"], int))
        await ws.close()

    await ws_c.close()