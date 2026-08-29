"""File-related tests.

Covers the file websocket message types and the HTTP upload endpoint:
  * get_user_files
  * upload (HTTP chunked endpoint)
  * delete_file
  * bulk_delete_files
  * convert_video (invalid-file error path)

Uploads are done over HTTP (``/upload/{uuid}`` + ``/upload/success/{uuid}``),
not the websocket. Several tests need a second websocket connection to verify
that a ``file_update`` pulse reaches a connection other than the one that
triggered the upload.
"""

import asyncio
import os
import tempfile
import uuid

from .harness import check, check_in
from .helpers import (fetch_preview_status, generate_test_video, open_ws,
                      recv_until, send_and_wait, upload_file)
from .test_accounts import register_and_login


async def test_get_user_files_and_collections():
    """Fetch a user's files and collections, and verify missing-auth errors."""
    print("\n[test] get user files + collections")
    email, password = await register_and_login()
    ws = await open_ws()

    resp = await send_and_wait(ws, "get_user_files",
                               {"email": email, "password": password},
                               "get_user_files_response")
    check("get_user_files responds", resp is not None)
    if resp:
        check("files is a list", isinstance(resp["data"], list))

    resp2 = await send_and_wait(ws, "get_user_collections",
                                {"email": email, "password": password},
                                "get_user_collections_response")
    check("get_user_collections responds", resp2 is not None)
    if resp2:
        check("collections is a list", isinstance(resp2["data"], list))

    # Missing credentials must error.
    resp3 = await send_and_wait(ws, "get_user_files", {},
                                "get_user_files_response")
    check("get_user_files missing auth errors", resp3 is not None)
    if resp3:
        check_in("missing auth error", "error", resp3["data"])

    await ws.close()


async def test_upload_and_file_pulse():
    """Upload a file and verify a ``file_update`` pulse reaches another connection.

    Sequence:
      1. Connection A authenticates (so it receives pulses).
      2. Connection B authenticates and performs the upload.
      3. Connection A should receive a ``file_update`` pulse for the new file.
    """
    print("\n[test] upload + file pulse to a second connection")
    email, password = await register_and_login()

    # Connection A: authenticate so it receives pulses.
    ws_a = await open_ws()
    await send_and_wait(ws_a, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")

    # Connection B: does the upload.
    ws_b = await open_ws()
    await send_and_wait(ws_b, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")

    up = await upload_file(email=email, password=password, filename="pulse.txt",
                           content=b"pulse test content")
    check("upload succeeds", up is not None)

    # Connection A should receive a file_update pulse for the new file.
    pulse = await recv_until(ws_a, lambda m: m.get("type") == "file_update")
    check("file_update pulse received on other conn", pulse is not None)
    if pulse:
        check("pulse toggle true", pulse["data"].get("toggle") is True)
        # NOTE: FileUpdate.File has no json tag, so Go serializes it as "File".
        check("pulse file has directory", bool(pulse["data"].get("File", {}).get("file_directory")))

    await ws_a.close()
    await ws_b.close()


async def test_delete_file():
    """Upload a file, delete it, then verify deleting it again errors."""
    print("\n[test] delete file")
    email, password = await register_and_login()
    ws = await open_ws()
    await send_and_wait(ws, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")

    up = await upload_file(email=email, password=password, filename="todelete.txt",
                           content=b"delete me")
    check("upload for delete succeeds", up is not None)
    if not up:
        await ws.close()
        return
    file_dir = up["fileDirectory"]

    resp = await send_and_wait(ws, "delete_file",
                               {"file_directory": file_dir,
                                "auth": {"email": email, "password": password}},
                               "delete_file_response")
    check("delete_file responds", resp is not None)
    if resp:
        check("delete success message", "success" in resp["data"])

    # Deleting again should error (file gone).
    resp2 = await send_and_wait(ws, "delete_file",
                                {"file_directory": file_dir,
                                 "auth": {"email": email, "password": password}},
                                "delete_file_response")
    check("delete missing file errors", resp2 is not None)
    if resp2:
        check_in("delete missing error", "error", resp2["data"])

    await ws.close()


async def test_bulk_delete_files():
    """Upload two files and delete them both in a single bulk request."""
    print("\n[test] bulk delete files")
    email, password = await register_and_login()
    ws = await open_ws()

    up1 = await upload_file(email=email, password=password, filename="b1.txt", content=b"one")
    up2 = await upload_file(email=email, password=password, filename="b2.txt", content=b"two")
    check("bulk setup uploads", up1 is not None and up2 is not None)
    if not (up1 and up2):
        await ws.close()
        return

    resp = await send_and_wait(ws, "bulk_delete_files",
                               {"file_directories": [up1["fileDirectory"], up2["fileDirectory"]],
                                "auth": {"email": email, "password": password}},
                               "bulk_delete_files_response")
    check("bulk delete responds", resp is not None)
    if resp:
        check("bulk deleted both", sorted(resp["data"].get("deleted", []))
              == sorted([up1["fileDirectory"], up2["fileDirectory"]]))
        check("bulk no errors", resp["data"].get("errors", []) == [])

    await ws.close()


async def test_convert_video_invalid():
    """Request a video conversion for a non-existent file; expect an error.

    Error responses use type ``"error"`` (only login keeps its own type on
    error), so we wait for ``"error"`` rather than ``convert_video_response``.
    """
    print("\n[test] convert video (invalid file)")
    email, password = await register_and_login()
    ws = await open_ws()

    resp = await send_and_wait(ws, "convert_video",
                               {"file_directory": "does_not_exist.txt",
                                "auth": {"email": email, "password": password}},
                               "error")
    check("convert missing file errors", resp is not None)
    if resp:
        check_in("convert error", "record not found", resp["data"])

    await ws.close()


async def test_video_preview_generation():
    """Upload a video and verify its GIF preview is generated within 1 minute.

    Sequence:
      1. Generate a small MP4 with ffmpeg.
      2. Upload it.
      3. Request the preview (expect 425 Too Early the first time).
      4. Poll the preview endpoint until it returns 200 (generated), failing
         if it is not generated within 60 seconds.
    """
    print("\n[test] video preview generation")
    email, password = await register_and_login()

    # Generate a small test video. The seed makes the content unique per run so
    # the preview is not already cached from a previous run.
    with tempfile.NamedTemporaryFile(suffix=".mp4", delete=False) as tmp:
        video_path = tmp.name
    try:
        seed = uuid.uuid4().hex[:8]
        generated = generate_test_video(video_path, duration=3, size="320x240", fps=24, seed=seed)
        check("ffmpeg generated test video", generated)
        if not generated:
            return
        with open(video_path, "rb") as f:
            content = f.read()
    finally:
        os.unlink(video_path)

    up = await upload_file(email=email, password=password, filename="preview.mp4",
                           content=content)
    check("upload video succeeds", up is not None)
    if not up:
        return
    file_dir = up["fileDirectory"]

    # First request should return 425 (Too Early) since the preview is not
    # generated yet, and it enqueues a runner.
    first_status = await fetch_preview_status(file_dir)
    check("first preview request returns 425", first_status == 425, f"(got {first_status})")

    # Poll until the preview is generated (200) or 60s elapse.
    generated_ok = False
    try:
        async with asyncio.timeout(60):
            while True:
                status = await fetch_preview_status(file_dir)
                if status == 200:
                    generated_ok = True
                    break
                await asyncio.sleep(1)
    except asyncio.TimeoutError:
        generated_ok = False

    check("video preview generated within 60s", generated_ok)


async def test_corrupted_video_preview_marker():
    """Verify a corrupted video gets an empty GIF marker (not retried forever).

    Sequence:
      1. Upload a file with a .mp4 extension but invalid (non-video) content.
      2. Request the preview (expect 425 the first time).
      3. The runner should fail to generate a real preview and instead write an
         empty GIF marker, so subsequent requests return 200 (the marker is
         served) and the runner never retries the corrupted file.
    """
    print("\n[test] corrupted video preview marker")
    email, password = await register_and_login()

    # Invalid "video" content (not a real mp4).
    up = await upload_file(email=email, password=password, filename="corrupt.mp4",
                           content=b"this is not a real video file")
    check("upload corrupted video succeeds", up is not None)
    if not up:
        return
    file_dir = up["fileDirectory"]

    # First request enqueues a runner and returns 425.
    first_status = await fetch_preview_status(file_dir)
    check("first corrupted preview request returns 425", first_status == 425, f"(got {first_status})")

    # The runner should fail and write an empty GIF marker, so the endpoint
    # eventually serves it (200) instead of retrying forever.
    marker_ok = False
    try:
        async with asyncio.timeout(60):
            while True:
                status = await fetch_preview_status(file_dir)
                if status == 200:
                    marker_ok = True
                    break
                await asyncio.sleep(1)
    except asyncio.TimeoutError:
        marker_ok = False

    check("corrupted video gets a preview marker (200)", marker_ok)


async def test_upload_updates_all_user_websockets():
    """Verify an upload sends a ``file_update`` pulse to ALL of the user's websockets.

    Sequence:
      1. The user opens two websocket connections (A and B), both authenticated.
      2. A third connection (C) performs the upload.
      3. Both A and B should receive a ``file_update`` pulse for the new file.
    """
    print("\n[test] upload updates all user websockets")
    email, password = await register_and_login()

    # Two websocket connections for the same user, both authenticated.
    user_conns = []
    for _ in range(2):
        ws = await open_ws()
        await send_and_wait(ws, "get_user_files",
                            {"email": email, "password": password},
                            "get_user_files_response")
        user_conns.append(ws)

    # A third connection performs the upload.
    ws_c = await open_ws()
    await send_and_wait(ws_c, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")
    up = await upload_file(email=email, password=password, filename="multi_ws.txt",
                           content=b"multi websocket pulse")
    check("upload succeeds", up is not None)

    # Both user connections should receive a file_update pulse.
    for i, ws in enumerate(user_conns):
        pulse = await recv_until(ws, lambda m: m.get("type") == "file_update")
        check(f"user conn {i} received file_update", pulse is not None)
        if pulse:
            check(f"user conn {i} toggle true", pulse["data"].get("toggle") is True)
            check(f"user conn {i} has file dir",
                  bool(pulse["data"].get("File", {}).get("file_directory")))
        await ws.close()

    await ws_c.close()