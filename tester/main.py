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

import argparse
import asyncio
import json
import sys

from . import config
from .harness import check, finish_test, start_test, summary
from .helpers import close_open_websockets, open_ws
from .registry import discover_tests, find_test


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


def parse_args(argv=None):
    parser = argparse.ArgumentParser(description="Run AngaDrive WebSocket integration tests")
    parser.add_argument("--test", help="run exactly one test by its stable name")
    parser.add_argument("--list", action="store_true", help="list available test names")
    parser.add_argument("--json", action="store_true", help="format --list output as JSON")
    return parser.parse_args(argv)


async def main(argv=None):
    args = parse_args(argv)
    tests = discover_tests()

    if args.list:
        names = [test.name for test in tests]
        print(json.dumps(names) if args.json else "\n".join(names))
        return 0

    if args.test:
        selected = find_test(args.test)
        if selected is None:
            print(f"ERROR: unknown integration test: {args.test}", file=sys.stderr)
            print("Available tests:", file=sys.stderr)
            print("\n".join(test.name for test in tests), file=sys.stderr)
            return 2
        tests = [selected]

    await _sanity_check_server()

    results = []
    for test in tests:
        start_test(test.name)
        try:
            async with asyncio.timeout(config.TEST_TIMEOUT):
                await test.function()
        except Exception as e:
            check(f"{test.name} raised", False, f"(exception: {e!r})")
        finally:
            await close_open_websockets()
        results.append(finish_test())

    return 0 if summary(results) else 1


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))