"""AngaDrive integration test runner.

Entry point for the modular test suite. Discovers and runs every test function
in the ``tester`` package, reports detailed pass/fail output, and exits with a
non-zero code if any assertion failed.

Usage:
    python -m tester.main                 # run against localhost:8080
    API_URL=myhost:8080 python -m tester.main

Requirements:
    pip install websockets aiohttp bcrypt
    The Go server must be running (go run .) with a reachable database.
"""

import asyncio
import sys

from . import config
from .harness import check, exit_with_result, summary
from .helpers import open_ws
from . import test_accounts, test_collections, test_files, test_homepage, test_misc

# Ordered list of test functions to run. Add new tests here to include them.
TESTS = [
    test_misc.test_unknown_message_type,
    test_accounts.register_and_login,
    test_accounts.test_change_display_name,
    test_accounts.test_change_email,
    test_accounts.test_change_password,
    test_files.test_get_user_files_and_collections,
    test_collections.test_collection_lifecycle,
    test_collections.test_collection_folder_and_file_ops,
    test_files.test_upload_and_file_pulse,
    test_files.test_upload_updates_all_user_websockets,
    test_files.test_upload_missing_auth,
    test_files.test_upload_invalid_credentials,
    test_files.test_upload_missing_filename,
    test_files.test_upload_missing_chunks,
    test_files.test_upload_invalid_total_chunks,
    test_files.test_upload_multiple_chunks,
    test_files.test_upload_empty_file,
    test_files.test_upload_large_file,
    test_files.test_upload_duplicate_content_dedup,
    test_files.test_delete_file,
    test_files.test_bulk_delete_files,
    test_files.test_convert_video_invalid,
    test_files.test_video_preview_generation,
    test_files.test_preview_serves_valid_gif,
    test_files.test_preview_unknown_file_returns_404,
    test_files.test_preview_dedup_no_duplicate_jobs,
    test_files.test_preview_serves_cached_after_generation,
    test_files.test_corrupted_video_preview_marker,
    test_misc.test_import_from_github_invalid,
    test_homepage.test_enable_homepage_updates,
    test_homepage.test_homepage_pulse_on_upload,
    test_homepage.test_homepage_pulse_reaches_all_subscribers,
    test_accounts.test_delete_account,
]


async def _sanity_check_server():
    """Verify the server is reachable before running any tests."""
    print(f"Connecting to {config.WS_URL}")
    try:
        ws = await open_ws()
        await ws.close()
    except Exception as e:
        print(f"ERROR: could not connect to {config.WS_URL}: {e}")
        print("Is the Go server running? (go run .)")
        sys.exit(1)


async def main():
    await _sanity_check_server()

    for t in TESTS:
        try:
            await t()
        except Exception as e:
            check(f"{t.__name__} raised", False, f"(exception: {e!r})")

    exit_with_result(summary())


if __name__ == "__main__":
    asyncio.run(main())