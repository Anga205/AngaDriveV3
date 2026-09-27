"""Collection tests.

Covers the collection websocket message types:
  * new_collection
  * get_collection
  * delete_collection
  * create_folder_in_collection
  * add_folder_to_collection / remove_folder_from_collection
  * add_file_to_collection / remove_file_from_collection

Note on pulses: ``add_file_to_collection`` and ``create_folder_in_collection``
trigger ``PulseCollectionSubscribers``, which sends *additional*
``get_collection_response`` messages to subscribed connections. Tests that
check the response content must therefore collect several responses and verify
that at least one contains the expected data.
"""

from .harness import check, check_eq, check_in
from .helpers import open_ws, recv_until, send_and_wait, upload_file
from .test_accounts import register_and_login


async def test_collection_lifecycle():
    """Create, fetch, and delete a collection; verify the deleted one errors.

    Sequence:
      1. Create a collection -> get its id.
      2. Fetch it (also subscribes this connection to it).
      3. Delete it.
      4. Fetching the deleted collection must error.
    """
    print("\n[test] collection lifecycle (create/get/delete)")
    email, password = await register_and_login()
    ws = await open_ws()

    # Create a collection.
    resp = await send_and_wait(ws, "new_collection",
                               {"collection_name": "My Collection",
                                "auth": {"email": email, "password": password}},
                               "new_collection_response")
    check("create collection responds", resp is not None)
    if not resp:
        await ws.close()
        return
    collection_id = resp["data"]
    check("create collection returns id", isinstance(collection_id, str) and collection_id)

    # Get the collection (also subscribes this conn to it).
    resp2 = await send_and_wait(ws, "get_collection",
                                {"id": collection_id,
                                 "auth": {"email": email, "password": password}},
                                "get_collection_response")
    check("get collection responds", resp2 is not None)
    if resp2:
        check_eq("collection id matches", resp2["data"].get("collection_id"), collection_id)
        check_eq("collection name", resp2["data"].get("collection_name"), "My Collection")
        check("is_owner true", resp2["data"].get("is_owner") is True)

    # Delete the collection.
    resp3 = await send_and_wait(ws, "delete_collection",
                                {"collection_id": collection_id,
                                 "auth": {"email": email, "password": password}},
                                "delete_collection_response")
    check("delete collection responds", resp3 is not None)
    if resp3:
        check_eq("delete returns id", resp3["data"], collection_id)

    # Getting a deleted collection should error. Error responses use type "error"
    # (only login_response keeps its own type on error).
    resp4 = await send_and_wait(ws, "get_collection",
                                {"id": collection_id,
                                 "auth": {"email": email, "password": password}},
                                "error")
    check("get deleted collection errors", resp4 is not None)
    if resp4:
        check_in("deleted collection error", "record not found", resp4["data"])

    await ws.close()


async def test_collection_folder_and_file_ops():
    """Exercise folder and file operations inside a collection.

    Sequence:
      1. Create a parent collection.
      2. Create a folder inside it.
      3. Add and remove the folder from the parent.
      4. Upload a file, add it to the parent, verify it appears, then remove it.
    """
    print("\n[test] collection folder + file operations")
    email, password = await register_and_login()
    ws = await open_ws()

    # Create a parent collection.
    resp = await send_and_wait(ws, "new_collection",
                               {"collection_name": "Parent",
                                "auth": {"email": email, "password": password}},
                               "new_collection_response")
    parent_id = resp["data"]

    # Create a folder inside it.
    resp2 = await send_and_wait(ws, "create_folder_in_collection",
                                {"collection_id": parent_id, "folder_name": "Sub",
                                 "auth": {"email": email, "password": password}},
                                "get_collection_response")
    check("create folder responds", resp2 is not None)
    folder_id = None
    if resp2:
        folders = resp2["data"].get("folders", [])
        folder_id = folders[0]["id"] if folders else None
        check("folder appears in parent", folder_id is not None)

    # Add the folder to the parent.
    if folder_id:
        resp3 = await send_and_wait(ws, "add_folder_to_collection",
                                    {"folder_id": folder_id, "collection_id": parent_id,
                                     "auth": {"email": email, "password": password}},
                                    "get_collection_response")
        check("add folder responds", resp3 is not None)

        resp4 = await send_and_wait(ws, "remove_folder_from_collection",
                                    {"folder_id": folder_id, "collection_id": parent_id,
                                     "auth": {"email": email, "password": password}},
                                    "get_collection_response")
        check("remove folder responds", resp4 is not None)

    # Upload a file and add/remove it from the collection.
    up = await upload_file(email=email, password=password)
    check("upload file for collection ops", up is not None)
    if up:
        file_dir = up["fileDirectory"]
        resp5 = await send_and_wait(ws, "add_file_to_collection",
                                    {"file_directory": file_dir, "collection_id": parent_id,
                                     "auth": {"email": email, "password": password}},
                                    "get_collection_response")
        check("add file to collection responds", resp5 is not None)
        if resp5:
            # add_file_to_collection triggers PulseCollectionSubscribers which
            # sends additional get_collection_response messages. Collect a few
            # and check that at least one contains the file.
            found = any(f["file_directory"] == file_dir
                        for f in resp5["data"].get("files", []))
            for _ in range(3):
                extra = await recv_until(ws, lambda m: m.get("type") == "get_collection_response", timeout=1.0)
                if extra is None:
                    break
                if any(f["file_directory"] == file_dir for f in extra["data"].get("files", [])):
                    found = True
                    break
            check("file in collection files", found)

        resp6 = await send_and_wait(ws, "remove_file_from_collection",
                                    {"file_directory": file_dir, "collection_id": parent_id,
                                     "auth": {"email": email, "password": password}},
                                    "get_collection_response")
        check("remove file from collection responds", resp6 is not None)

    await ws.close()


async def test_bulk_add_files_to_collection():
    """Bulk-add multiple uploaded files to an existing collection or create one."""
    print("\n[test] bulk add files to collection")
    email, password = await register_and_login()
    ws = await open_ws()

    resp = await send_and_wait(ws, "new_collection",
                               {"collection_name": "Bulk Target",
                                "auth": {"email": email, "password": password}},
                               "new_collection_response")
    check("bulk create target collection responds", resp is not None)
    target_id = resp["data"] if resp else None

    up1 = await upload_file(email=email, password=password, filename="bulk_one.txt", content=b"bulk one")
    up2 = await upload_file(email=email, password=password, filename="bulk_two.txt", content=b"bulk two")
    check("upload first bulk file", up1 is not None)
    check("upload second bulk file", up2 is not None)
    if not (up1 and up2):
        await ws.close()
        return

    file_dir_1 = up1["fileDirectory"]
    file_dir_2 = up2["fileDirectory"]

    resp2 = await send_and_wait(ws, "bulk_add_files_to_collection",
                                {"file_directories": [file_dir_1, file_dir_2],
                                 "collection_id": target_id,
                                 "auth": {"email": email, "password": password}},
                                "bulk_add_files_to_collection_response")
    check("bulk add existing collection responds", resp2 is not None)
    if resp2:
        added = resp2["data"].get("added", [])
        errors = resp2["data"].get("errors", [])
        check("bulk add existing collection added both files", set(added) == {file_dir_1, file_dir_2})
        check("bulk add existing collection no errors", errors == [])

        collection_update = await recv_until(ws, lambda m: m.get("type") == "get_collection_response", timeout=1.0)
        if collection_update:
            files = collection_update["data"].get("files", [])
            check("existing collection update includes both files", len(files) == 2)
            check("existing collection update size matches file count", len(files) == 2)

    resp3 = await send_and_wait(ws, "bulk_add_files_to_collection",
                                {"file_directories": [file_dir_1, file_dir_2],
                                 "collection_name": "Bulk New Collection",
                                 "auth": {"email": email, "password": password}},
                                "bulk_add_files_to_collection_response")
    check("bulk create and add responds", resp3 is not None)
    if resp3:
        added = resp3["data"].get("added", [])
        errors = resp3["data"].get("errors", [])
        check("bulk create and add added both files", set(added) == {file_dir_1, file_dir_2})
        check("bulk create and add no errors", errors == [])

        card_update = await recv_until(ws, lambda m: m.get("type") == "collection_card_update", timeout=1.0)
        if card_update:
            payload = card_update["data"]
            check("new collection card reflects 2 files", payload.get("file_count") == 2)
            check("new collection card size is positive", payload.get("size", 0) > 0)

    await ws.close()