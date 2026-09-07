"""
AngaDrive integration test suite.

This package contains a set of standalone Python scripts (NOT pytest) that
connect to the live AngaDrive websocket + HTTP endpoints, send known requests,
and validate the responses against expected behaviour.

Layout:
    config.py          - connection configuration (URLs, timeouts)
    harness.py         - the pass/fail reporting harness
    helpers.py         - shared helpers (websocket, upload, auth)
    test_accounts.py   - account lifecycle tests (register/login/change/delete)
    test_files.py      - file tests (get/upload/delete/bulk/convert)
    test_collections.py- collection tests (create/get/delete/folder/file ops)
    test_homepage.py   - homepage pulse tests
    test_misc.py       - misc tests (unknown type, github import)
    registry.py        - automatic discovery of async test functions
    main.py            - entry point for full or single-test runs

Run from the repo root:
    python -m tester.main
"""