"""Discovery of independently runnable WebSocket integration tests."""

import inspect
import importlib
import pkgutil
from dataclasses import dataclass

import tester


@dataclass(frozen=True)
class TestCase:
    """A stable name and coroutine for one integration-test execution."""

    name: str
    function: object


def discover_tests():
    """Discover public async test functions across ``tester/test_*.py``."""
    tests = []
    module_names = sorted(
        info.name for info in pkgutil.iter_modules(tester.__path__)
        if info.name.startswith("test_")
    )
    for module_name in module_names:
        module = importlib.import_module(f"{tester.__name__}.{module_name}")
        for name, function in inspect.getmembers(module, inspect.iscoroutinefunction):
            if name.startswith("test_"):
                tests.append(TestCase(name, function))
    return tests


def find_test(name):
    """Return a test by its stable name, or ``None`` when it is unknown."""
    return next((test for test in discover_tests() if test.name == name), None)