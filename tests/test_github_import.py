"""End-to-end GitHub repository import test.

This test verifies that importing a GitHub repository creates a faithful mirror
of the repository inside AngaDrive:

  * the root collection is named after the repository,
  * every folder in the repository becomes a nested collection,
  * every file in the repository is added to the matching collection,
  * every imported file is downloadable and byte-identical to the repository.

The test is opt-in because it needs network access and ``git``, and it clones a
real repository. Enable it with ``GITHUB_IMPORT_TEST=1``.

Sequence:
  1. Clone the repository locally (``git clone --depth 1``).
  2. Build the expected folder/file tree from the local clone.
  3. Register a fresh account and trigger ``import_from_github``.
  4. Recursively walk the imported collections and compare them to the clone.
  5. Fetch every imported file and compare its bytes to the clone.
  6. Clean up: delete the imported files and collections, remove the clone.
"""

import asyncio
import os
import shutil
import subprocess
import tempfile
import time

from . import config
from .harness import check, check_eq
from .helpers import http_get, open_ws, send_and_recv_until, send_and_wait
from .test_accounts import register_and_login

# Set to "1" to run this test. It is skipped otherwise.
OPT_IN_ENV = "GITHUB_IMPORT_TEST"

# Repository imported by default; override with GITHUB_IMPORT_REPO.
DEFAULT_REPO_URL = "https://github.com/Anga205/AngaDriveV3"


def _clone_repo(repo_url, dest):
    """Clone ``repo_url`` into ``dest`` (an existing empty directory)."""
    try:
        subprocess.run(
            ["git", "clone", "--depth", "1", repo_url, dest],
            check=True, capture_output=True, timeout=300,
        )
        return True
    except (FileNotFoundError, subprocess.CalledProcessError,
            subprocess.TimeoutExpired) as exc:
        print(f"  [SKIP-CAUSE] git clone failed: {exc}")
        return False


def _expected_tree(root):
    """Build ``{relative_dir: {"files": set, "dirs": set}}`` from a local clone.

    Uses ``os.scandir`` with ``follow_symlinks=False`` so the semantics match the
    server's ``filepath.WalkDir`` (which uses lstat). The ``.git`` directory is
    skipped because the server removes it before importing.
    """
    tree = {}

    def scan(path, rel):
        files, dirs = set(), set()
        with os.scandir(path) as entries:
            for entry in entries:
                if entry.name == ".git":
                    continue
                if entry.is_dir(follow_symlinks=False):
                    dirs.add(entry.name)
                    scan(entry.path, f"{rel}/{entry.name}" if rel else entry.name)
                else:
                    files.add(entry.name)
        tree[rel] = {"files": files, "dirs": dirs}

    scan(root, "")
    return tree


async def _get_user_collections(ws, auth):
    """Return the user's top-level collection cards."""
    resp = await send_and_wait(ws, "get_user_collections", auth,
                               "get_user_collections_response")
    return (resp or {}).get("data", []) or []


async def _wait_for_root_collection(ws, auth, timeout=30):
    """Poll until the user has at least one collection (the imported root)."""
    deadline = time.monotonic() + timeout
    while True:
        cards = await _get_user_collections(ws, auth)
        if cards or time.monotonic() >= deadline:
            return cards
        await asyncio.sleep(0.5)


async def _get_collection(ws, collection_id, auth):
    """Fetch one collection, ignoring unrelated pulse responses."""
    return await send_and_recv_until(
        ws, "get_collection", {"id": collection_id, "auth": auth},
        lambda m: (m.get("type") == "get_collection_response"
                   and isinstance(m.get("data"), dict)
                   and m["data"].get("collection_id") == collection_id),
        timeout=config.DEFAULT_TIMEOUT,
    )


async def _verify_collection(ws, collection_id, rel_path, expected, clone_root,
                             auth, collected_files, collected_collections):
    """Recursively compare one AngaDrive collection to the local clone."""
    label = rel_path or "<root>"
    resp = await _get_collection(ws, collection_id, auth)
    check(f"{label}: collection responds", resp is not None)
    if not resp:
        return

    data = resp["data"]
    depth = rel_path.count("/") if rel_path else 0
    collected_collections.append((depth, collection_id))

    actual_files = {f["original_file_name"] for f in data.get("files", [])}
    actual_dirs = {f["name"] for f in data.get("folders", [])}
    want = expected.get(rel_path, {"files": set(), "dirs": set()})

    check(f"{label}: file set matches repo", actual_files == want["files"],
          f"missing={sorted(want['files'] - actual_files)} "
          f"extra={sorted(actual_files - want['files'])}")
    check(f"{label}: folder set matches repo", actual_dirs == want["dirs"],
          f"missing={sorted(want['dirs'] - actual_dirs)} "
          f"extra={sorted(actual_dirs - want['dirs'])}")

    # Every imported file must be downloadable and byte-identical to the clone.
    for file_entry in data.get("files", []):
        file_dir = file_entry["file_directory"]
        name = file_entry["original_file_name"]
        collected_files.append(file_dir)
        local_path = os.path.join(clone_root, rel_path, name)
        status, body = await http_get(f"{config.HTTP_URL}/i/{file_dir}")
        check(f"{label}/{name}: served", status == 200, f"status {status}")
        if status == 200 and os.path.exists(local_path):
            with open(local_path, "rb") as handle:
                local_bytes = handle.read()
            check(f"{label}/{name}: bytes match repo", body == local_bytes,
                  f"{len(body)} bytes served vs {len(local_bytes)} local")

    # Recurse into folders that exist in the repository.
    folder_ids = {f["name"]: f["id"] for f in data.get("folders", [])}
    for name in sorted(want["dirs"]):
        child_id = folder_ids.get(name)
        if child_id is None:
            continue  # already reported by the folder-set assertion
        child_rel = f"{rel_path}/{name}" if rel_path else name
        await _verify_collection(ws, child_id, child_rel, expected, clone_root,
                                 auth, collected_files, collected_collections)


async def _cleanup(ws, auth, collected_files, collected_collections):
    """Best-effort removal of the imported files and collections."""
    if auth is None:
        return

    # Delete imported files in batches to keep each request small.
    for start in range(0, len(collected_files), 100):
        batch = collected_files[start:start + 100]
        try:
            await send_and_wait(ws, "bulk_delete_files",
                                {"file_directories": batch, "auth": auth},
                                "bulk_delete_files_response")
        except Exception as exc:  # noqa: BLE001 - cleanup must never fail the test
            print(f"  [cleanup] bulk delete failed: {exc!r}")

    # Delete child collections deepest-first, then the root.
    for _, collection_id in sorted(collected_collections, reverse=True):
        try:
            await send_and_wait(ws, "delete_collection",
                                {"collection_id": collection_id, "auth": auth},
                                "delete_collection_response")
        except Exception as exc:  # noqa: BLE001 - cleanup must never fail the test
            print(f"  [cleanup] delete collection failed: {exc!r}")


async def test_github_import_full_tree():
    """Import a real GitHub repo and verify every folder and file recursively."""
    if os.getenv(OPT_IN_ENV) != "1":
        print(f"  [SKIP] set {OPT_IN_ENV}=1 to run the full GitHub import test")
        return

    repo_url = os.getenv("GITHUB_IMPORT_REPO", DEFAULT_REPO_URL)
    repo_name = repo_url.rstrip("/").split("/")[-1]

    clone_dir = tempfile.mkdtemp(prefix="angadrive_import_")
    auth = None
    ws = None
    collected_files = []
    collected_collections = []
    try:
        cloned = _clone_repo(repo_url, clone_dir)
        check("local clone succeeds", cloned, f"git clone {repo_url} failed",
              server=None)
        if not cloned:
            return

        expected = _expected_tree(clone_dir)
        check("local clone contains files",
              any(entry["files"] for entry in expected.values()), server=None)

        email, password = await register_and_login()
        auth = {"email": email, "password": password}
        ws = await open_ws()

        resp = await send_and_wait(ws, "import_from_github",
                                   {"repo_url": repo_url, "auth": auth},
                                   "success_notification",
                                   timeout=config.GITHUB_IMPORT_TIMEOUT)
        check("import_from_github responds", resp is not None,
              f"no success_notification within {config.GITHUB_IMPORT_TIMEOUT}s")
        if not resp:
            return
        check("import returns the clone path",
              isinstance(resp.get("data"), str) and bool(resp["data"]))

        cards = await _wait_for_root_collection(ws, auth)
        check_eq("exactly one root collection after import", len(cards), 1)
        root = cards[0] if cards else None
        check("root collection is named after the repo",
              root is not None and root["name"] == repo_name,
              f"expected a collection named {repo_name!r}")
        if root is None:
            return

        await _verify_collection(ws, root["id"], "", expected, clone_dir, auth,
                                 collected_files, collected_collections)
    finally:
        if ws is not None:
            await _cleanup(ws, auth, collected_files, collected_collections)
            await ws.close()
        shutil.rmtree(clone_dir, ignore_errors=True)
