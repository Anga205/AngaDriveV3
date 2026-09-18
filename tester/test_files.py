"""File-related tests.

Covers the file websocket message types and the HTTP upload endpoint:
  * get_user_files
  * upload (HTTP chunked endpoint)
  * delete_file
  * bulk_delete_files
  * convert_video (invalid-file error path)
  * convert_image (to-PNG conversion, to-JPG conversion, invalid-file error path)

Uploads are done over HTTP (``/upload/{uuid}`` + ``/upload/success/{uuid}``),
not the websocket. Several tests need a second websocket connection to verify
that a ``file_update`` pulse reaches a connection other than the one that
triggered the upload.
"""

import asyncio
import os
import tempfile
import uuid

from . import config
from .harness import check, check_in
from .helpers import (fetch_file, fetch_preview, fetch_preview_status,
                      finalize_upload, generate_test_image, generate_test_video,
                      make_msg, open_ws, recv_until, send_and_wait, send_chunk,
                      upload_file, upload_file_full)
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


async def test_rename_file():
    """Rename metadata without changing content, ownership, or file identity."""
    print("\n[test] rename file")
    email, password = await register_and_login()
    other_email, other_password = await register_and_login()
    ws_a = await open_ws()
    ws_b = await open_ws()
    await send_and_wait(ws_a, "get_user_files", {"email": email, "password": password}, "get_user_files_response")
    await send_and_wait(ws_b, "get_user_files", {"email": email, "password": password}, "get_user_files_response")

    content = b"rename without reupload"
    up = await upload_file(email=email, password=password, filename="before.txt", content=content)
    check("rename setup upload succeeds", up is not None)
    if not up:
        await ws_a.close()
        await ws_b.close()
        return
    file_dir = up["fileDirectory"]

    # Drain the upload pulse on the second connection before testing rename.
    await recv_until(ws_b, lambda m: m.get("type") == "file_update" and
                     m.get("data", {}).get("File", {}).get("file_directory") == file_dir)

    import aiohttp
    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/download/{file_dir}") as response:
            original_content = await response.read()
            original_size = len(original_content)
    check("downloaded content matches before rename", original_content == content)

    renamed = "renamed file with extension.txt"
    resp = await send_and_wait(ws_a, "rename_file",
                               {"file_directory": file_dir,
                                "new_file_name": renamed,
                                "auth": {"email": email, "password": password}},
                               "rename_file_response")
    check("rename responds", resp is not None)
    if resp:
        check("rename succeeds", "success" in resp["data"])

    pulse = await recv_until(ws_b, lambda m: m.get("type") == "file_update" and
                             m.get("data", {}).get("replace") is True and
                             m.get("data", {}).get("File", {}).get("file_directory") == file_dir)
    check("rename pulse reaches second websocket", pulse is not None)
    if pulse:
        check("rename pulse has new filename", pulse["data"]["File"]["original_file_name"] == renamed)

    files = await send_and_wait(ws_a, "get_user_files",
                                {"email": email, "password": password},
                                "get_user_files_response")
    renamed_file = next((file for file in (files or {}).get("data", [])
                         if file.get("file_directory") == file_dir), None)
    check("renamed file remains in user list", renamed_file is not None)
    if renamed_file:
        check("new filename is persisted", renamed_file["original_file_name"] == renamed)
        check("old filename is absent", renamed_file["original_file_name"] != "before.txt")
        check("file directory is unchanged", renamed_file["file_directory"] == file_dir)
        check("file size is unchanged", renamed_file["file_size"] == original_size)

    # Repeated renames, long names, and invalid names use the same file identity.
    long_name = "x" * 200 + ".txt"
    for new_name in (long_name, "final-name.md"):
        repeated = await send_and_wait(ws_a, "rename_file",
                                       {"file_directory": file_dir,
                                        "new_file_name": new_name,
                                        "auth": {"email": email, "password": password}},
                                       "rename_file_response")
        check(f"rename to {new_name[:12]} succeeds", repeated is not None and "success" in repeated["data"])

    for invalid_name in ("", ".", "../escape.txt", "bad\\name.txt"):
        invalid = await send_and_wait(ws_a, "rename_file",
                                      {"file_directory": file_dir,
                                       "new_file_name": invalid_name,
                                       "auth": {"email": email, "password": password}},
                                      "rename_file_response")
        check(f"invalid filename {invalid_name!r} is rejected",
              invalid is not None and "error" in invalid["data"])

    missing = await send_and_wait(ws_a, "rename_file",
                                  {"file_directory": "missing-file-directory",
                                   "new_file_name": "missing.txt",
                                   "auth": {"email": email, "password": password}},
                                  "rename_file_response")
    check("renaming nonexistent file fails", missing is not None and "error" in missing["data"])

    unauthenticated = await send_and_wait(ws_a, "rename_file",
                                          {"file_directory": file_dir,
                                           "new_file_name": "unauthenticated.txt",
                                           "auth": {}},
                                          "rename_file_response")
    check("unauthenticated rename fails", unauthenticated is not None and "error" in unauthenticated["data"])

    invalid_auth = await send_and_wait(ws_a, "rename_file",
                                       {"file_directory": file_dir,
                                        "new_file_name": "invalid-auth.txt",
                                        "auth": {"email": email, "password": "wrong"}},
                                       "rename_file_response")
    check("invalid authentication rename fails", invalid_auth is not None and "error" in invalid_auth["data"])

    ws_other = await open_ws()
    unauthorized = await send_and_wait(ws_other, "rename_file",
                                       {"file_directory": file_dir,
                                        "new_file_name": "other-user.txt",
                                        "auth": {"email": other_email, "password": other_password}},
                                       "rename_file_response")
    check("other user cannot rename file", unauthorized is not None and "error" in unauthorized["data"])
    await ws_other.close()

    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/download/{file_dir}") as response:
            renamed_content = await response.read()
    check("file content is unchanged", renamed_content == original_content)
    check("file size is unchanged after repeated renames", len(renamed_content) == original_size)

    await ws_a.close()
    await ws_b.close()


async def test_duplicate_file():
    """Duplicate a file and verify its content and metadata are independent."""
    print("\n[test] duplicate file")
    email, password = await register_and_login()
    ws = await open_ws()

    content = b"duplicate file content"
    up = await upload_file(email=email, password=password, filename="original.txt", content=content)
    check("duplicate setup upload succeeds", up is not None)
    if not up:
        await ws.close()
        return
    original_dir = up["fileDirectory"]

    response = await send_and_wait(
        ws, "duplicate_file",
        {"file_directory": original_dir,
         "auth": {"email": email, "password": password}},
        "duplicate_file_response",
    )
    check("duplicate_file responds", response is not None)
    check("duplicate_file succeeds", response is not None and "success" in response["data"])

    files = await send_and_wait(ws, "get_user_files",
                                {"email": email, "password": password},
                                "get_user_files_response")
    entries = (files or {}).get("data", [])
    duplicate = next((file for file in entries
                      if file.get("file_directory") != original_dir and
                      file.get("original_file_name") == "Copy of original.txt"), None)
    original = next((file for file in entries
                     if file.get("file_directory") == original_dir), None)
    check("duplicate has a new file ID", duplicate is not None and duplicate["file_directory"] != original_dir)
    check("original and duplicate are separately persisted", original is not None and duplicate is not None)

    import aiohttp
    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/download/{original_dir}") as response:
            original_content = await response.read()
        async with session.get(f"{config.HTTP_URL}/download/{duplicate['file_directory']}") as response:
            duplicate_content = await response.read()
    check("duplicate retains the same SHA256 content", duplicate_content == original_content == content)

    delete_response = await send_and_wait(
        ws, "delete_file",
        {"file_directory": original_dir,
         "auth": {"email": email, "password": password}},
        "delete_file_response",
    )
    check("original can be deleted independently", delete_response is not None and "success" in delete_response["data"])
    if duplicate:
        async with aiohttp.ClientSession() as session:
            async with session.get(f"{config.HTTP_URL}/download/{duplicate['file_directory']}") as response:
                remaining_content = await response.read()
        check("duplicate remains after original deletion", remaining_content == content)

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


async def test_upload_missing_auth():
    """Finalize an upload without credentials; expect 400."""
    print("\n[test] upload missing auth")
    import uuid as _uuid
    upload_id = _uuid.uuid4().hex
    status, body = await finalize_upload(upload_id, 1, "noauth.txt")
    check("upload without auth returns 400", status == 400, f"(got {status})")
    if isinstance(body, str):
        check_in("missing auth message", "Missing authentication", body)


async def test_upload_invalid_credentials():
    """Finalize an upload with wrong credentials; expect 401."""
    print("\n[test] upload invalid credentials")
    import uuid as _uuid
    upload_id = _uuid.uuid4().hex
    status, body = await finalize_upload(upload_id, 1, "badauth.txt",
                                         email="nobody@example.com", password="wrong")
    check("upload with bad credentials returns 401", status == 401, f"(got {status})")


async def test_upload_missing_filename():
    """Finalize an upload with no originalFileName; expect 400."""
    print("\n[test] upload missing filename")
    email, password = await register_and_login()
    import uuid as _uuid
    upload_id = _uuid.uuid4().hex
    status, body = await finalize_upload(upload_id, 1, "",
                                         email=email, password=password)
    check("upload without filename returns 400", status == 400, f"(got {status})")


async def test_upload_missing_chunks():
    """Finalize an upload where some chunks were never sent; expect 400."""
    print("\n[test] upload missing chunks")
    email, password = await register_and_login()
    import uuid as _uuid
    upload_id = _uuid.uuid4().hex

    # Send only chunk 0 of a 3-chunk upload.
    await send_chunk(upload_id, 0, b"only chunk zero")

    status, body = await finalize_upload(upload_id, 3, "partial.txt",
                                         email=email, password=password)
    check("upload with missing chunks returns 400", status == 400, f"(got {status})")
    if isinstance(body, dict):
        missing = body.get("missingChunks", [])
        check("missing chunks reported", missing == [1, 2], f"(got {missing})")


async def test_upload_invalid_total_chunks():
    """Finalize an upload with a non-numeric totalChunks; expect 400."""
    print("\n[test] upload invalid total chunks")
    email, password = await register_and_login()
    import uuid as _uuid
    upload_id = _uuid.uuid4().hex
    await send_chunk(upload_id, 0, b"data")

    # Send totalChunks as a non-integer string.
    import aiohttp
    async with aiohttp.ClientSession() as session:
        form = aiohttp.FormData()
        form.add_field("totalChunks", "not-a-number")
        form.add_field("originalFileName", "bad.txt")
        form.add_field("email", email)
        form.add_field("password", password)
        async with session.post(f"{config.HTTP_URL}/upload/success/{upload_id}", data=form) as resp:
            status = resp.status
    check("upload with invalid totalChunks returns 400", status == 400, f"(got {status})")


async def test_upload_multiple_chunks():
    """Upload a file split across multiple chunks and verify it assembles.

    Sequence:
      1. Split content into 4 chunks and send each.
      2. Finalize with totalChunks=4.
      3. Verify the upload succeeds and returns a file_directory.
    """
    print("\n[test] upload multiple chunks")
    email, password = await register_and_login()

    content = b"multi-chunk-content-" * 100  # ~2000 bytes
    status, body = await upload_file_full(email=email, password=password,
                                          filename="multi.txt", content=content,
                                          total_chunks=4)
    check("multi-chunk upload succeeds", status == 200, f"(got {status})")
    if isinstance(body, dict):
        check("multi-chunk upload returns file_directory", bool(body.get("fileDirectory")))


async def test_upload_empty_file():
    """Upload an empty file (zero bytes) and verify it succeeds."""
    print("\n[test] upload empty file")
    email, password = await register_and_login()

    status, body = await upload_file_full(email=email, password=password,
                                          filename="empty.txt", content=b"")
    check("empty file upload succeeds", status == 200, f"(got {status})")
    if isinstance(body, dict):
        check("empty file upload returns file_directory", bool(body.get("fileDirectory")))


async def test_upload_large_file():
    """Upload a larger file (a few MB) and verify it succeeds."""
    print("\n[test] upload large file")
    email, password = await register_and_login()

    # ~3 MB of pseudo-random-ish content.
    content = os.urandom(3 * 1024 * 1024)
    status, body = await upload_file_full(email=email, password=password,
                                          filename="large.bin", content=content,
                                          chunk_size=512 * 1024)
    check("large file upload succeeds", status == 200, f"(got {status})")
    if isinstance(body, dict):
        check("large file upload returns file_directory", bool(body.get("fileDirectory")))


async def test_upload_duplicate_content_dedup():
    """Upload two files with identical content; verify both succeed.

    The server content-addresses files by sha256, so two uploads of the same
    bytes should both succeed (they share the same on-disk file but get
    distinct file_directories).
    """
    print("\n[test] upload duplicate content")
    email, password = await register_and_login()

    content = b"identical-content-for-dedup-test"
    up1 = await upload_file(email=email, password=password, filename="dup_a.txt",
                            content=content)
    up2 = await upload_file(email=email, password=password, filename="dup_b.txt",
                            content=content)
    check("first duplicate upload succeeds", up1 is not None)
    check("second duplicate upload succeeds", up2 is not None)
    if up1 and up2:
        check("duplicate uploads get distinct directories",
              up1["fileDirectory"] != up2["fileDirectory"])


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


async def test_convert_image_to_png():
    """Upload a JPEG and verify it is converted to a lossless PNG.

    Sequence:
      1. Generate a small JPEG with Pillow.
      2. Upload it.
      3. Send ``convert_image`` and wait for ``convert_image_response``.
      4. Assert the response carries a new file whose name ends in ``.png``.
      5. Fetch the converted file and assert it starts with the PNG magic bytes.
    """
    print("\n[test] convert image to png")
    email, password = await register_and_login()
    ws = await open_ws()
    await send_and_wait(ws, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")

    with tempfile.NamedTemporaryFile(suffix=".jpg", delete=False) as tmp:
        image_path = tmp.name
    try:
        seed = uuid.uuid4().hex[:8]
        generated = generate_test_image(image_path, fmt="jpeg", size=(64, 64), seed=seed)
        check("pillow generated test image", generated,
              "Pillow is not installed; install it with 'pip install pillow'")
        if not generated:
            await ws.close()
            return
        with open(image_path, "rb") as f:
            content = f.read()
    finally:
        os.unlink(image_path)

    up = await upload_file(email=email, password=password, filename="photo.jpg",
                           content=content)
    check("upload jpeg succeeds", up is not None)
    if not up:
        await ws.close()
        return
    file_dir = up["fileDirectory"]

    # The handler replies immediately with the file directory string, then the
    # async runner sends a second convert_image_response carrying the new file.
    # Wait for the async one that contains the converted file object.
    await ws.send(make_msg("convert_image",
                           {"file_directory": file_dir,
                            "auth": {"email": email, "password": password}}))
    resp = await recv_until(
        ws,
        lambda m: (m.get("type") == "convert_image_response"
                   and isinstance(m.get("data"), dict)
                   and m["data"].get("file") is not None),
        timeout=config.DEFAULT_TIMEOUT,
    )
    check("convert_image responds", resp is not None)
    if not resp:
        await ws.close()
        return

    check("convert_image has no error", not resp["data"].get("error"),
          f"unexpected error: {resp['data'].get('error')}")
    converted = resp["data"].get("file")
    check("convert_image returns a file", converted is not None)
    if converted:
        check("converted file is a png",
              converted.get("original_file_name", "").lower().endswith(".png"),
              f"got {converted.get('original_file_name')}")
        check("converted file has a directory", bool(converted.get("file_directory")))

        # Fetch the converted file and verify PNG magic bytes.
        status, body = await fetch_file(converted["file_directory"])
        check("converted file is served", status == 200, f"status {status}")
        check("converted file has png magic bytes", body.startswith(b"\x89PNG\r\n\x1a\n"),
              f"first bytes: {body[:8]!r}")

    await ws.close()


async def test_convert_image_invalid():
    """Request an image conversion for a non-existent file; expect an error.

    Error responses use type ``"error"`` (only login keeps its own type on
    error), so we wait for ``"error"`` rather than ``convert_image_response``.
    """
    print("\n[test] convert image (invalid file)")
    email, password = await register_and_login()
    ws = await open_ws()

    resp = await send_and_wait(ws, "convert_image",
                               {"file_directory": "does_not_exist.jpg",
                                "auth": {"email": email, "password": password}},
                               "error")
    check("convert image missing file errors", resp is not None)
    if resp:
        check_in("convert image error", "record not found", resp["data"])

    await ws.close()


async def test_convert_png_to_jpg():
    """Upload a PNG and verify it is converted to a small JPEG.

    Sequence:
      1. Generate a small PNG with Pillow.
      2. Upload it.
      3. Send ``convert_image`` and wait for the async ``convert_image_response``.
      4. Assert the response carries a new file whose name ends in ``.jpg``.
      5. Fetch the converted file and assert it starts with JPEG magic bytes.
    """
    print("\n[test] convert png to jpg")
    email, password = await register_and_login()
    ws = await open_ws()
    await send_and_wait(ws, "get_user_files",
                        {"email": email, "password": password},
                        "get_user_files_response")

    with tempfile.NamedTemporaryFile(suffix=".png", delete=False) as tmp:
        image_path = tmp.name
    try:
        seed = uuid.uuid4().hex[:8]
        generated = generate_test_image(image_path, fmt="png", size=(64, 64), seed=seed)
        check("pillow generated test png", generated,
              "Pillow is not installed; install it with 'pip install pillow'")
        if not generated:
            await ws.close()
            return
        with open(image_path, "rb") as f:
            content = f.read()
    finally:
        os.unlink(image_path)

    up = await upload_file(email=email, password=password, filename="photo.png",
                           content=content)
    check("upload png succeeds", up is not None)
    if not up:
        await ws.close()
        return
    file_dir = up["fileDirectory"]

    # The handler replies immediately with the file directory string, then the
    # async runner sends a second convert_image_response carrying the new file.
    await ws.send(make_msg("convert_image",
                           {"file_directory": file_dir,
                            "auth": {"email": email, "password": password}}))
    resp = await recv_until(
        ws,
        lambda m: (m.get("type") == "convert_image_response"
                   and isinstance(m.get("data"), dict)
                   and m["data"].get("file") is not None),
        timeout=config.DEFAULT_TIMEOUT,
    )
    check("convert_image responds", resp is not None)
    if not resp:
        await ws.close()
        return

    check("convert_image has no error", not resp["data"].get("error"),
          f"unexpected error: {resp['data'].get('error')}")
    converted = resp["data"].get("file")
    check("convert_image returns a file", converted is not None)
    if converted:
        check("converted file is a jpg",
              converted.get("original_file_name", "").lower().endswith(".jpg"),
              f"got {converted.get('original_file_name')}")
        check("converted file has a directory", bool(converted.get("file_directory")))

        # Fetch the converted file and verify JPEG magic bytes.
        status, body = await fetch_file(converted["file_directory"])
        check("converted file is served", status == 200, f"status {status}")
        check("converted file has jpeg magic bytes", body.startswith(b"\xff\xd8\xff"),
              f"first bytes: {body[:8]!r}")

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
        check("ffmpeg generated test video", generated,
              "ffmpeg is not installed or failed; install it with "
              "'sudo apt-get install -y ffmpeg' (CI installs it automatically)")
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


async def _upload_generated_video(email, password, filename="preview.mp4",
                                  duration=3, size="320x240", fps=24):
    """Generate a unique test video, upload it, and return the file_directory.

    Returns ``None`` if ffmpeg is unavailable or the upload fails.
    """
    with tempfile.NamedTemporaryFile(suffix=".mp4", delete=False) as tmp:
        video_path = tmp.name
    try:
        seed = uuid.uuid4().hex[:8]
        generated = generate_test_video(video_path, duration=duration, size=size,
                                        fps=fps, seed=seed)
        if not generated:
            print("  [SKIP-CAUSE] ffmpeg is not installed or failed; install it "
                  "with 'sudo apt-get install -y ffmpeg' (CI installs it "
                  "automatically)")
            return None
        with open(video_path, "rb") as f:
            content = f.read()
    finally:
        os.unlink(video_path)

    up = await upload_file(email=email, password=password, filename=filename,
                           content=content)
    if not up:
        return None
    return up["fileDirectory"]


async def _wait_for_preview(file_dir, timeout=60):
    """Poll the preview endpoint until it returns 200; return the body bytes.

    Returns ``(True, body)`` on success or ``(False, None)`` on timeout.
    """
    try:
        async with asyncio.timeout(timeout):
            while True:
                status, body = await fetch_preview(file_dir)
                if status == 200:
                    return True, body
                await asyncio.sleep(1)
    except asyncio.TimeoutError:
        return False, None


async def test_preview_serves_valid_gif():
    """Verify a generated preview is a valid GIF with the expected magic bytes.

    Sequence:
      1. Upload a generated video.
      2. Wait for the preview to be generated (200).
      3. Assert the served body starts with the GIF89a magic bytes and is
         non-trivial in size (a real preview, not the empty marker).
    """
    print("\n[test] preview serves valid gif")
    email, password = await register_and_login()

    file_dir = await _upload_generated_video(email, password)
    check("upload generated video succeeds", file_dir is not None)
    if not file_dir:
        return

    ok, body = await _wait_for_preview(file_dir)
    check("preview generated within 60s", ok)
    if not ok:
        return

    check("preview body is non-empty", len(body) > 0)
    check("preview starts with GIF89a magic", body[:6] == b"GIF89a",
          f"(got {body[:6]!r})")
    # A real preview should be larger than the 33-byte empty marker.
    check("preview is a real gif (not empty marker)", len(body) > 33,
          f"(got {len(body)} bytes)")


async def test_preview_unknown_file_returns_404():
    """Request a preview for a file that does not exist; expect 404."""
    print("\n[test] preview unknown file returns 404")
    status, _ = await fetch_preview("does_not_exist.mp4")
    check("unknown file preview returns 404", status == 404, f"(got {status})")


async def test_preview_dedup_no_duplicate_jobs():
    """Verify repeated preview requests do not enqueue duplicate jobs.

    Sequence:
      1. Upload a generated video.
      2. Fire several preview requests in quick succession (all before the
         preview is generated).
      3. The first returns 425; the preview should still be generated exactly
         once and eventually served (200). Because jobs are deduplicated by
         sha256, the extra requests must not break generation.
    """
    print("\n[test] preview dedup no duplicate jobs")
    email, password = await register_and_login()

    file_dir = await _upload_generated_video(email, password)
    check("upload generated video succeeds", file_dir is not None)
    if not file_dir:
        return

    # Fire several requests immediately; the first enqueues, the rest should
    # be deduplicated (all 425 until generated).
    statuses = []
    for _ in range(5):
        statuses.append(await fetch_preview_status(file_dir))

    check("all early preview requests return 425", all(s == 425 for s in statuses),
          f"(got {statuses})")

    ok, body = await _wait_for_preview(file_dir)
    check("preview still generated after dedup requests", ok)
    if ok:
        check("dedup preview is a real gif", len(body) > 33, f"(got {len(body)} bytes)")


async def test_preview_serves_cached_after_generation():
    """Verify a generated preview is served from cache on subsequent requests.

    Sequence:
      1. Upload a generated video and wait for its preview (200).
      2. Request the preview again; it should still return 200 with the same
         body (served from disk, not regenerated).
    """
    print("\n[test] preview served from cache after generation")
    email, password = await register_and_login()

    file_dir = await _upload_generated_video(email, password)
    check("upload generated video succeeds", file_dir is not None)
    if not file_dir:
        return

    ok, first_body = await _wait_for_preview(file_dir)
    check("preview generated within 60s", ok)
    if not ok:
        return

    status, second_body = await fetch_preview(file_dir)
    check("second preview request returns 200", status == 200, f"(got {status})")
    check("cached preview body identical", second_body == first_body)


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