"""Shared helpers for the AngaDrive integration tests.

These functions abstract the common operations every test needs:

  * opening a websocket connection,
  * sending a request and waiting for a specific response type,
  * waiting for a message matching a predicate (used for pulses),
  * uploading a file over the HTTP chunked-upload endpoint,
  * generating bcrypt hashes and random emails.

The helpers intentionally return ``None`` on timeout/failure so tests can
assert on the result directly.
"""

import asyncio
import gzip
import json
import os
import subprocess
import uuid

import websockets

from . import config

OPEN_WEBSOCKETS = set()


def make_msg(msg_type, data):
    """Build a websocket message in the ``{"type": ..., "data": ...}`` shape."""
    return json.dumps({"type": msg_type, "data": data})


def bcrypt_hash(password):
    """Return a bcrypt hash of ``password`` (matches Go's bcrypt usage).

    The Go server stores a bcrypt hash at registration and compares it against
    the plaintext password at login, so tests must send a hash when registering
    and the plaintext when logging in.
    """
    import bcrypt
    return bcrypt.hashpw(password.encode(), bcrypt.gensalt()).decode()


def random_email():
    """Return a unique test email address."""
    return f"tester_{uuid.uuid4().hex[:10]}@example.com"


async def open_ws():
    """Open a websocket connection to the server."""
    ws = await websockets.connect(config.WS_URL, open_timeout=config.DEFAULT_TIMEOUT)
    OPEN_WEBSOCKETS.add(ws)
    return ws


async def close_open_websockets():
    """Close connections left behind by a failed test."""
    connections = list(OPEN_WEBSOCKETS)
    OPEN_WEBSOCKETS.clear()
    if connections:
        await asyncio.gather(*(ws.close() for ws in connections), return_exceptions=True)


async def recv_until(ws, predicate, timeout=config.DEFAULT_TIMEOUT):
    """Read messages until one satisfies ``predicate``; return it or None.

    Args:
        ws:        the websocket connection.
        predicate: a callable taking a parsed message dict and returning bool.
        timeout:   seconds to wait before giving up.

    Returns the first matching message dict, or ``None`` on timeout/close.
    """
    try:
        async with asyncio.timeout(timeout):
            while True:
                raw = await ws.recv()
                try:
                    msg = json.loads(raw)
                except json.JSONDecodeError:
                    continue
                if predicate(msg):
                    return msg
    except (asyncio.TimeoutError, websockets.ConnectionClosed):
        return None


async def send_and_wait(ws, msg_type, data, want_type, timeout=config.DEFAULT_TIMEOUT):
    """Send a request and wait for the response of the given type.

    Args:
        ws:        the websocket connection.
        msg_type:  the request message type (e.g. ``"login"``).
        data:      the request payload.
        want_type: the response type to wait for (e.g. ``"login_response"``).
        timeout:   seconds to wait.

    Returns the matching response dict, or ``None`` on timeout.
    """
    await ws.send(make_msg(msg_type, data))
    return await recv_until(ws, lambda m: m.get("type") == want_type, timeout)


async def upload_file(email=None, password=None, token=None, collection_id="",
                      filename="hello.txt", content=b"hello world"):
    """Upload a small file via the chunked HTTP upload endpoint.

    The upload endpoint is HTTP (not websocket). It requires authentication via
    either ``token`` OR ``email`` + ``password``.

    Args:
        email/password/token: authentication credentials.
        collection_id:        optional collection to add the file to.
        filename:             the original file name.
        content:              the raw file bytes.

    Returns the JSON response body (dict) or ``None`` on failure.
    """
    import aiohttp

    upload_id = uuid.uuid4().hex
    chunk = gzip.compress(content)

    async with aiohttp.ClientSession() as session:
        # 1. Send the single chunk (gzipped).
        data = aiohttp.FormData()
        data.add_field("chunk", chunk, filename="chunk", content_type="application/octet-stream")
        data.add_field("chunkIndex", "0")
        async with session.post(f"{config.HTTP_URL}/upload/{upload_id}", data=data) as resp:
            if resp.status != 200:
                return None

        # 2. Finalize the upload.
        form = aiohttp.FormData()
        form.add_field("totalChunks", "1")
        form.add_field("originalFileName", filename)
        form.add_field("collectionId", collection_id)
        if token:
            form.add_field("token", token)
        if email:
            form.add_field("email", email)
        if password:
            form.add_field("password", password)
        async with session.post(f"{config.HTTP_URL}/upload/success/{upload_id}", data=form) as resp:
            if resp.status != 200:
                return None
            return await resp.json()


def generate_test_video(path, duration=3, size="320x240", fps=24, seed=None):
    """Generate a small test MP4 video using ffmpeg.

    ``seed`` (if given) is drawn into the frame so that videos generated with
    different seeds have different content (and therefore different sha256),
    which keeps preview tests isolated across runs.

    Returns True on success, False if ffmpeg is unavailable or fails.
    """
    try:
        # Draw the seed text into the frame so content differs per seed.
        draw = ""
        if seed is not None:
            draw = f",drawtext=text='{seed}':fontsize=24:fontcolor=white:x=10:y=10"
        subprocess.run(
            ["ffmpeg", "-y", "-f", "lavfi", "-i", f"testsrc=duration={duration}:size={size}:rate={fps}",
             "-vf", f"format=yuv420p{draw}",
             "-pix_fmt", "yuv420p", path],
            check=True, capture_output=True,
        )
        return True
    except (FileNotFoundError, subprocess.CalledProcessError):
        return False


def generate_test_image(path, fmt="jpeg", size=(64, 64), seed=None):
    """Generate a small non-PNG test image using Pillow.

    ``seed`` (if given) is drawn into the image so that images generated with
    different seeds have different content (and therefore different sha256),
    which keeps conversion tests isolated across runs.

    Returns True on success, False if Pillow is unavailable or fails.
    """
    try:
        from PIL import Image, ImageDraw
    except ImportError:
        return False

    try:
        img = Image.new("RGB", size, (30, 40, 50))
        draw = ImageDraw.Draw(img)
        if seed is not None:
            draw.text((4, 4), seed, fill=(255, 255, 255))
        draw.rectangle([8, 8, size[0] - 8, size[1] - 8], outline=(200, 120, 40), width=2)
        img.save(path, format=fmt.upper())
        return True
    except Exception:
        return False


async def fetch_preview_status(file_directory):
    """Request a video preview and return the HTTP status code.

    Returns the status code (int), or None on connection error.
    """
    import aiohttp
    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/preview-video/{file_directory}.gif") as resp:
            return resp.status


async def fetch_preview(file_directory):
    """Request a video preview and return ``(status, body_bytes)``.

    Unlike :func:`fetch_preview_status`, this also returns the response body so
    tests can inspect the served content (e.g. verify GIF magic bytes).
    """
    import aiohttp
    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/preview-video/{file_directory}.gif") as resp:
            body = await resp.read()
            return resp.status, body


async def fetch_file(file_directory):
    """Request a stored file and return ``(status, body_bytes)``.

    Files are served at ``/i/{file_directory}``. Used to verify the bytes of a
    converted file (e.g. PNG magic bytes).
    """
    import aiohttp
    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/i/{file_directory}") as resp:
            body = await resp.read()
            return resp.status, body


async def send_chunk(upload_id, chunk_index, content):
    """Send a single gzipped chunk to ``/upload/{upload_id}``.

    Returns the HTTP status code (int), or None on connection error.
    """
    import aiohttp
    chunk = gzip.compress(content)
    async with aiohttp.ClientSession() as session:
        data = aiohttp.FormData()
        data.add_field("chunk", chunk, filename="chunk", content_type="application/octet-stream")
        data.add_field("chunkIndex", str(chunk_index))
        async with session.post(f"{config.HTTP_URL}/upload/{upload_id}", data=data) as resp:
            return resp.status


async def finalize_upload(upload_id, total_chunks, filename, email=None,
                          password=None, token=None, collection_id=""):
    """Finalize an upload and return ``(status, body)``.

    This is the low-level counterpart to :func:`upload_file`; it returns the
    raw HTTP status and parsed JSON body (or raw text) so tests can assert on
    error codes and messages.
    """
    import aiohttp
    async with aiohttp.ClientSession() as session:
        form = aiohttp.FormData()
        form.add_field("totalChunks", str(total_chunks))
        form.add_field("originalFileName", filename)
        form.add_field("collectionId", collection_id)
        if token:
            form.add_field("token", token)
        if email:
            form.add_field("email", email)
        if password:
            form.add_field("password", password)
        async with session.post(f"{config.HTTP_URL}/upload/success/{upload_id}", data=form) as resp:
            text = await resp.text()
            try:
                body = json.loads(text)
            except json.JSONDecodeError:
                body = text
            return resp.status, body


async def upload_file_full(email=None, password=None, token=None, collection_id="",
                           filename="hello.txt", content=b"hello world",
                           total_chunks=1, chunk_size=None):
    """Upload a file via the chunked endpoint, returning ``(status, body)``.

    Splits ``content`` into ``total_chunks`` (or into ``chunk_size``-byte
    pieces) and sends each chunk before finalizing. Returns the finalize
    response as ``(status, body)``.
    """
    import aiohttp

    upload_id = uuid.uuid4().hex

    # Split content into chunks.
    if chunk_size is not None:
        pieces = [content[i:i + chunk_size] for i in range(0, len(content), chunk_size)] or [b""]
    else:
        # Split into total_chunks roughly equal pieces.
        n = max(1, total_chunks)
        base = len(content) // n
        rem = len(content) % n
        pieces = []
        idx = 0
        for i in range(n):
            size = base + (1 if i < rem else 0)
            pieces.append(content[idx:idx + size])
            idx += size

    async with aiohttp.ClientSession() as session:
        for i, piece in enumerate(pieces):
            chunk = gzip.compress(piece)
            data = aiohttp.FormData()
            data.add_field("chunk", chunk, filename="chunk", content_type="application/octet-stream")
            data.add_field("chunkIndex", str(i))
            async with session.post(f"{config.HTTP_URL}/upload/{upload_id}", data=data) as resp:
                if resp.status != 200:
                    return resp.status, None

        form = aiohttp.FormData()
        form.add_field("totalChunks", str(len(pieces)))
        form.add_field("originalFileName", filename)
        form.add_field("collectionId", collection_id)
        if token:
            form.add_field("token", token)
        if email:
            form.add_field("email", email)
        if password:
            form.add_field("password", password)
        async with session.post(f"{config.HTTP_URL}/upload/success/{upload_id}", data=form) as resp:
            text = await resp.text()
            try:
                body = json.loads(text)
            except json.JSONDecodeError:
                body = text
            return resp.status, body