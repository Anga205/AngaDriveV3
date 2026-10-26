/**
 * Typed websocket operations. Each function wraps a single backend message type
 * so components never construct raw envelopes.
 */
import { getAuth } from "./auth";
import { request, send, WsResult } from "./client";

export function getUserFiles(ws: WebSocket | undefined): boolean {
  return send(ws, "get_user_files", getAuth());
}

export function getUserCollections(ws: WebSocket | undefined): boolean {
  return send(ws, "get_user_collections", getAuth());
}

export function getCollection(ws: WebSocket | undefined, id: string): boolean {
  return send(ws, "get_collection", { id, auth: getAuth() });
}

export function renameFile(ws: WebSocket | undefined, fileDirectory: string, newFileName: string): boolean {
  return send(ws, "rename_file", { file_directory: fileDirectory, new_file_name: newFileName, auth: getAuth() });
}

export function duplicateFile(ws: WebSocket | undefined, fileDirectory: string): boolean {
  return send(ws, "duplicate_file", { file_directory: fileDirectory, auth: getAuth() });
}

export function deleteFile(ws: WebSocket | undefined, fileDirectory: string): boolean {
  return send(ws, "delete_file", { file_directory: fileDirectory, auth: getAuth() });
}

export function convertVideo(ws: WebSocket | undefined, fileDirectory: string): boolean {
  return send(ws, "convert_video", { file_directory: fileDirectory, auth: getAuth() });
}

export function convertImage(ws: WebSocket | undefined, fileDirectory: string, targetFormat: string): boolean {
  return send(ws, "convert_image", { file_directory: fileDirectory, target_format: targetFormat, auth: getAuth() });
}

export function removeFileFromCollection(ws: WebSocket | undefined, fileDirectory: string, collectionId: string): boolean {
  return send(ws, "remove_file_from_collection", { file_directory: fileDirectory, collection_id: collectionId, auth: getAuth() });
}

export function addFileToCollection(ws: WebSocket | undefined, fileDirectory: string, collectionId: string): boolean {
  return send(ws, "add_file_to_collection", { file_directory: fileDirectory, collection_id: collectionId, auth: getAuth() });
}

export function addFolderToCollection(ws: WebSocket | undefined, folderId: string, collectionId: string): boolean {
  return send(ws, "add_folder_to_collection", { folder_id: folderId, collection_id: collectionId, auth: getAuth() });
}

export function removeFolderFromCollection(ws: WebSocket | undefined, folderId: string, collectionId: string): boolean {
  return send(ws, "remove_folder_from_collection", { folder_id: folderId, collection_id: collectionId, auth: getAuth() });
}

export function createFolderInCollection(ws: WebSocket | undefined, collectionId: string, folderName: string): boolean {
  return send(ws, "create_folder_in_collection", { collection_id: collectionId, folder_name: folderName, auth: getAuth() });
}

export function createCollection(ws: WebSocket | undefined, collectionName: string): boolean {
  return send(ws, "new_collection", { collection_name: collectionName, auth: getAuth() });
}

export function deleteCollection(ws: WebSocket | undefined, collectionId: string): boolean {
  return send(ws, "delete_collection", { collection_id: collectionId, auth: getAuth() });
}

export function bulkDeleteFiles(ws: WebSocket | undefined, fileDirectories: string[]): boolean {
  return send(ws, "bulk_delete_files", { file_directories: fileDirectories, auth: getAuth() });
}

export function bulkDuplicateFiles(ws: WebSocket | undefined, fileDirectories: string[]): boolean {
  return send(ws, "bulk_duplicate_files", { file_directories: fileDirectories, auth: getAuth() });
}

export function bulkAddFilesToCollection(
  ws: WebSocket | undefined,
  fileDirectories: string[],
  collectionId: string,
  collectionName = "",
): boolean {
  return send(ws, "bulk_add_files_to_collection", {
    file_directories: fileDirectories,
    collection_id: collectionId,
    collection_name: collectionName,
    auth: getAuth(),
  });
}

export function importFromGithub(ws: WebSocket | undefined, repoUrl: string): boolean {
  return send(ws, "import_from_github", { repo_url: repoUrl, auth: getAuth() });
}

export function enableHomepageUpdates(ws: WebSocket | undefined, enabled: boolean): boolean {
  return send(ws, "enable_homepage_updates", enabled);
}

export function deleteAccount(ws: WebSocket | undefined): boolean {
  return send(ws, "delete_account", getAuth());
}

export function changeDisplayName(ws: WebSocket | undefined, displayName: string): Promise<WsResult> {
  return request(ws!, "change_display_name", { display_name: displayName, email: getAuth().email, password: getAuth().password }, "change_display_name_response");
}

export function changeEmail(ws: WebSocket | undefined, oldEmail: string, newEmail: string): Promise<WsResult> {
  return request(ws!, "change_email", { old_email: oldEmail, new_email: newEmail, password: getAuth().password }, "change_email_response");
}

export function changePassword(ws: WebSocket | undefined, oldPassword: string, newPasswordHashed: string): Promise<WsResult> {
  return request(ws!, "change_password", { email: getAuth().email, old_password: oldPassword, new_password_hashed: newPasswordHashed }, "change_password_response");
}
