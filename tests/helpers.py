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
import time
import uuid

import websockets

from . import config
from .harness import record_steps

OPEN_WEBSOCKETS = set()


def make_msg(msg_type, data):
    """Build a websocket message in the ``{"type": ..., "data": ...}`` shape."""
    return json.dumps({"type": msg_type, "data": data})


def sha256_hex(password):
    """Return the lowercase hex SHA-256 digest of ``password``.

    The frontend sends this digest instead of the plaintext password, so tests
    must do the same for login and for every authenticated request.
    """
    import hashlib
    return hashlib.sha256(password.encode()).hexdigest()


def bcrypt_hash(password):
    """Return bcrypt(sha256(password)) (matches the Go server's storage).

    The Go server stores bcrypt of the hex SHA-256 digest at registration, so
    tests must send that value in ``hashed_password``.
    """
    import bcrypt
    return bcrypt.hashpw(sha256_hex(password).encode(), bcrypt.gensalt()).decode()


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


async def recv_until(ws, predicate, timeout=config.DEFAULT_TIMEOUT, record=True):
    """Read messages until one satisfies ``predicate``; return it or None.

    Args:
        ws:        the websocket connection.
        predicate: a callable taking a parsed message dict and returning bool.
        timeout:   seconds to wait before giving up.
        record:    when True (default), publish the round-trip time of the
                   matching message for the next assertion. Set False when the
                   caller records its own timing (e.g. :func:`send_and_wait`).

    Returns the first matching message dict, or ``None`` on timeout/close.
    """
    started = time.perf_counter()
    try:
        async with asyncio.timeout(timeout):
            while True:
                raw = await ws.recv()
                try:
                    msg = json.loads(raw)
                except json.JSONDecodeError:
                    continue
                if predicate(msg):
                    if record:
                        record_steps([("recv", time.perf_counter() - started)])
                    return msg
    except (asyncio.TimeoutError, websockets.ConnectionClosed):
        if record:
            record_steps([])
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
    started = time.perf_counter()
    await ws.send(make_msg(msg_type, data))
    response = await recv_until(ws, lambda m: m.get("type") == want_type,
                                timeout, record=False)
    if response is not None:
        record_steps([(msg_type, time.perf_counter() - started)])
    else:
        record_steps([])
    return response


async def send_and_recv_until(ws, msg_type, data, predicate,
                              timeout=config.DEFAULT_TIMEOUT):
    """Send a request and wait for the first message matching ``predicate``.

    Unlike :func:`send_and_wait`, the response type is not fixed: the caller
    supplies a predicate so it can wait for an asynchronous follow-up message
    (e.g. the second ``convert_image_response`` carrying the converted file).
    The measured time is the full send-to-response round trip.

    Returns the matching message dict, or ``None`` on timeout.
    """
    started = time.perf_counter()
    await ws.send(make_msg(msg_type, data))
    response = await recv_until(ws, predicate, timeout, record=False)
    if response is not None:
        record_steps([(msg_type, time.perf_counter() - started)])
    else:
        record_steps([])
    return response


async def recv_messages(ws, count, timeout=config.DEFAULT_TIMEOUT):
    """Read up to ``count`` messages and return the parsed list.

    Used to drain a burst of messages (e.g. the homepage update burst). The
    round-trip time to receive the burst is published for the next assertion.
    """
    started = time.perf_counter()
    messages = []
    try:
        async with asyncio.timeout(timeout):
            while len(messages) < count:
                raw = await ws.recv()
                try:
                    messages.append(json.loads(raw))
                except json.JSONDecodeError:
                    continue
    except (asyncio.TimeoutError, websockets.ConnectionClosed):
        pass
    if messages:
        record_steps([("recv burst", time.perf_counter() - started)])
    else:
        record_steps([])
    return messages


async def upload_file(email=None, password=None, token=None, collection_id="",
                      filename="hello.txt", content=b"hello world"):
    """Upload a small file via the chunked HTTP upload endpoint.

    The upload endpoint is HTTP (not websocket). It requires authentication via
    either ``token`` OR ``email`` + ``password``.

    Args:
        email/password/token: authentication credentials. ``password`` must be
                              the SHA-256 digest (see :func:`sha256_hex`), which
                              is what the frontend sends.
        collection_id:        optional collection to add the file to.
        filename:             the original file name.
        content:              the raw file bytes.

    Returns the JSON response body (dict) or ``None`` on failure.
    """
    import aiohttp

    upload_id = uuid.uuid4().hex
    chunk = gzip.compress(content)
    steps = []

    async with aiohttp.ClientSession() as session:
        # 1. Send the single chunk (gzipped).
        data = aiohttp.FormData()
        data.add_field("chunk", chunk, filename="chunk", content_type="application/octet-stream")
        data.add_field("chunkIndex", "0")
        started = time.perf_counter()
        async with session.post(f"{config.HTTP_URL}/upload/{upload_id}", data=data) as resp:
            if resp.status != 200:
                steps.append(("chunk POST", time.perf_counter() - started))
                record_steps(steps)
                return None
        steps.append(("chunk POST", time.perf_counter() - started))

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
        started = time.perf_counter()
        async with session.post(f"{config.HTTP_URL}/upload/success/{upload_id}", data=form) as resp:
            if resp.status != 200:
                steps.append(("finalize POST", time.perf_counter() - started))
                record_steps(steps)
                return None
            body = await resp.json()
        steps.append(("finalize POST", time.perf_counter() - started))

    record_steps(steps)
    return body


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


async def http_get(url):
    """GET ``url`` and return ``(status, body_bytes)``.

    Publishes the round-trip time for the next assertion.
    """
    import aiohttp
    started = time.perf_counter()
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as resp:
            body = await resp.read()
            status = resp.status
    record_steps([("GET", time.perf_counter() - started)])
    return status, body


async def http_post_form(url, fields):
    """POST a form to ``url`` and return ``(status, body)``.

    ``fields`` is a mapping of form field name to value. The body is parsed as
    JSON when possible, otherwise returned as text. Publishes the round-trip
    time for the next assertion.
    """
    import aiohttp
    started = time.perf_counter()
    async with aiohttp.ClientSession() as session:
        form = aiohttp.FormData()
        for key, value in fields.items():
            form.add_field(key, value)
        async with session.post(url, data=form) as resp:
            status = resp.status
            text = await resp.text()
    record_steps([("POST", time.perf_counter() - started)])
    try:
        body = json.loads(text)
    except json.JSONDecodeError:
        body = text
    return status, body


async def fetch_preview_status(file_directory):
    """Request a video preview and return the HTTP status code.

    Returns the status code (int), or None on connection error.
    """
    import aiohttp
    started = time.perf_counter()
    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/preview-video/{file_directory}.webp") as resp:
            status = resp.status
    record_steps([("preview GET", time.perf_counter() - started)])
    return status


async def fetch_preview(file_directory):
    """Request a video preview and return ``(status, body_bytes)``.

    Unlike :func:`fetch_preview_status`, this also returns the response body so
    tests can inspect the served content (e.g. verify WebP magic bytes).
    """
    import aiohttp
    started = time.perf_counter()
    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/preview-video/{file_directory}.webp") as resp:
            body = await resp.read()
            status = resp.status
    record_steps([("preview GET", time.perf_counter() - started)])
    return status, body


async def fetch_file(file_directory):
    """Request a stored file and return ``(status, body_bytes)``.

    Files are served at ``/i/{file_directory}``. Used to verify the bytes of a
    converted file (e.g. PNG magic bytes).
    """
    import aiohttp
    started = time.perf_counter()
    async with aiohttp.ClientSession() as session:
        async with session.get(f"{config.HTTP_URL}/i/{file_directory}") as resp:
            body = await resp.read()
            status = resp.status
    record_steps([("file GET", time.perf_counter() - started)])
    return status, body

async def fetch_named_file(file_directory, original_name):
    """Request a stored file via the copy-link URL form.

    The frontend builds copy links as
    ``/i/{file_directory-without-extension}/{original_name}``. This helper
    reproduces that URL so tests can verify the named-file route resolves the
    extension correctly. Returns ``(status, body_bytes)``.
    """
    import aiohttp
    base = file_directory.rsplit(".", 1)[0] if "." in file_directory else file_directory
    url = f"{config.HTTP_URL}/i/{base}/{original_name}"
    started = time.perf_counter()
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as resp:
            body = await resp.read()
            status = resp.status
    record_steps([("named file GET", time.perf_counter() - started)])
    return status, body

async def send_chunk(upload_id, chunk_index, content):
    """Send a single gzipped chunk to ``/upload/{upload_id}``.

    Returns the HTTP status code (int), or None on connection error.
    """
    import aiohttp
    chunk = gzip.compress(content)
    started = time.perf_counter()
    async with aiohttp.ClientSession() as session:
        data = aiohttp.FormData()
        data.add_field("chunk", chunk, filename="chunk", content_type="application/octet-stream")
        data.add_field("chunkIndex", str(chunk_index))
        async with session.post(f"{config.HTTP_URL}/upload/{upload_id}", data=data) as resp:
            status = resp.status
    record_steps([("chunk POST", time.perf_counter() - started)])
    return status


async def finalize_upload(upload_id, total_chunks, filename, email=None,
                          password=None, token=None, collection_id=""):
    """Finalize an upload and return ``(status, body)``.

    This is the low-level counterpart to :func:`upload_file`; it returns the
    raw HTTP status and parsed JSON body (or raw text) so tests can assert on
    error codes and messages.
    """
    import aiohttp
    started = time.perf_counter()
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
            status = resp.status
            text = await resp.text()
    record_steps([("finalize POST", time.perf_counter() - started)])
    try:
        body = json.loads(text)
    except json.JSONDecodeError:
        body = text
    return status, body


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

    steps = []
    async with aiohttp.ClientSession() as session:
        for i, piece in enumerate(pieces):
            chunk = gzip.compress(piece)
            data = aiohttp.FormData()
            data.add_field("chunk", chunk, filename="chunk", content_type="application/octet-stream")
            data.add_field("chunkIndex", str(i))
            started = time.perf_counter()
            async with session.post(f"{config.HTTP_URL}/upload/{upload_id}", data=data) as resp:
                status = resp.status
            steps.append((f"chunk {i} POST", time.perf_counter() - started))
            if status != 200:
                record_steps(steps)
                return status, None

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
        started = time.perf_counter()
        async with session.post(f"{config.HTTP_URL}/upload/success/{upload_id}", data=form) as resp:
            status = resp.status
            text = await resp.text()
        steps.append(("finalize POST", time.perf_counter() - started))

    record_steps(steps)
    try:
        body = json.loads(text)
    except json.JSONDecodeError:
        body = text
    return status, body