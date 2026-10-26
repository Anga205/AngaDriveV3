import { useContext } from "solid-js";
import { toast } from "solid-toast";
import { AppContext } from "@/Context";
import { useWebSocket } from "@/Websockets";
import { bulkAddFilesToCollection, bulkDeleteFiles, bulkDuplicateFiles } from "@/api/endpoints";

/**
 * Shared bulk file actions used by both the desktop and mobile drive views.
 * Centralizes the "selected files" plumbing and websocket availability checks.
 */
export function useBulkFileActions() {
  const ctx = useContext(AppContext)!;
  const { socket: getSocket } = useWebSocket();

  const selected = () => Array.from(ctx.selectedFiles?.() || new Set<string>());

  const clearSelection = () => ctx.setSelectedFiles?.(new Set());

  const duplicate = () => {
    const files = selected();
    if (files.length === 0) return;
    if (!bulkDuplicateFiles(getSocket(), files)) {
      toast.error("WebSocket is not available");
      return;
    }
    clearSelection();
  };

  const remove = () => {
    const files = selected();
    if (files.length === 0) return;
    if (!bulkDeleteFiles(getSocket(), files)) {
      toast.error("WebSocket is not available");
      return;
    }
    clearSelection();
  };

  const addToCollection = (collectionId: string, collectionName: string) => {
    const files = selected();
    if (files.length === 0) return;
    if (!bulkAddFilesToCollection(getSocket(), files, collectionId, collectionName)) {
      toast.error("WebSocket is not available");
      return;
    }
    clearSelection();
  };

  return { duplicate, remove, addToCollection, clearSelection };
}
