import SparkMD5 from 'spark-md5';
import toast from 'solid-toast';
import type { AppContextType, CollectionCardData, FileData, SocketStatus } from './types';
import { Accessor } from 'solid-js';

async function getFileMD5(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();

    reader.onload = function (event) {
      const arrayBuffer = event.target?.result;
      if (arrayBuffer instanceof ArrayBuffer) {
        const wordArray = SparkMD5.ArrayBuffer.hash(arrayBuffer);
        resolve(wordArray);
      } else {
        reject(new Error('Failed to read file as ArrayBuffer'));
      }
    };

    reader.onerror = function () {
      reject(new Error('Error reading file'));
    };

    reader.readAsArrayBuffer(file);
  });
}

const formatFileSize = (size: number) => {
    if (size < 1024) return `${size} B`;
    const kb = size / 1024;
    if (kb < 1024) return `${kb.toFixed(2)} KB`;
    const mb = kb / 1024;
    if (mb < 1024) return `${mb.toFixed(2)} MB`;
    const gb = mb / 1024;
    return `${gb.toFixed(2)} GB`;
};

const truncateFileName = (name: string) => {
    return name.length > 32 ? `${name.slice(0, 32)}...` : name;
};

const getFileType = (filePath: string) => {
    const ext = filePath.split('.').pop()?.toLowerCase();
    if (!ext) return "Unknown";
    if (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext)) return "Image";
    if (["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(ext)) return "Video";
    if (["mp3", "wav", "aac", "flac", "ogg", "wma", "m4a"].includes(ext)) return "Audio";
    if (["pdf"].includes(ext)) return "Document";
    if (["doc", "docx", "odt", "rtf", "txt", "md", "tex"].includes(ext)) return "Text Document";
    if (["xls", "xlsx", "ods", "csv", "tsv"].includes(ext)) return "Spreadsheet";
    if (["ppt", "pptx", "odp", "key"].includes(ext)) return "Presentation";
    if (["zip", "rar", "7z", "tar", "gz", "bz2", "xz"].includes(ext)) return "Archive";
    if (["exe", "msi", "bat", "sh", "apk", "dmg"].includes(ext)) return "Executable";
    if (["html", "htm", "css", "js", "ts", "jsx", "tsx", "json", "xml", "yaml", "yml", "c", "py"].includes(ext)) return "Code";
    if (["iso", "img", "bin", "cue"].includes(ext)) return "Disk Image";
    if (["epub", "mobi", "azw", "azw3"].includes(ext)) return "Ebook";
    return "Unknown";
};

const UniversalMessageHandler = (message: MessageEvent, ctx: AppContextType) => {
  const data = JSON.parse(message.data);
  if (data.type === "get_user_files_response") {
      ctx.setFiles(data.data.sort((a: FileData, b: FileData) => {
        if (b.timestamp === a.timestamp) {
          return a.file_directory.localeCompare(b.file_directory);
        }
        return b.timestamp - a.timestamp;
      }) || []);
  } else if (data.type === "file_update") {
      if (data.data.toggle === true) {
          ctx.setFiles((prev: FileData[]) => [data.data.File, ...prev]);
      } else if (data.data.toggle === false) {
          ctx.setFiles((prev: FileData[]) => prev.filter((file: FileData) => file.file_directory !== data.data.File.file_directory));
      }
  } else if (data.type === "get_user_collections_response") {
      ctx.setUserCollections(data.data.sort((a: CollectionCardData, b: CollectionCardData) => {
        if (b.timestamp - a.timestamp === a.timestamp - b.timestamp) {
          return a.id.localeCompare(b.id);
        }
        return b.timestamp - a.timestamp;
      }) || []);
  } else if (data.type === "collection_update") {
      if (data.data.toggle === true) {
          console.log("Collection added:", data.data.collection);
          ctx.setUserCollections((prev: CollectionCardData[]) => [data.data.collection, ...prev]);
      } else if (data.data.toggle === false) {
          ctx.setUserCollections((prev: CollectionCardData[]) => prev.filter((collection: CollectionCardData) => collection.id !== data.data.collection.id));
          ctx.setKnownCollections(prev => {
            const newCollections = { ...prev };
            delete newCollections[data.data.collection.id];
            return newCollections;
          });
      }
  } else if (data.type === "error") {
      toast.error(`Error: ${data.data.error}`, {
        style: {
          "background-color": "#2a2a2a",
          "color": "#ffffff"
        }
      });
  } else if (data.type === "get_collection_response") {
    ctx.setKnownCollections(prev=>({
      ...prev,
      [data.data.collection_id]: {
        name: data.data.collection_name,
        files: data.data.files,
        folders: data.data.folders,
        isOwned: data.data.is_owner
      }
    }))
  }
}

function generateClientToken(): string {
    const chars = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890";
    const randomChoice = (arr: string, k: number): string => {
        let result = "";
        for (let i = 0; i < k; i++) {
            result += arr[Math.floor(Math.random() * arr.length)];
        }
        return result;
    };
    const part1 = randomChoice(chars, 10);
    const part2 = randomChoice(chars, 20);
    const part3 = String(Math.round(Date.now() / 1000));
    return `${part1}.${part2}.${part3}`;
}

const fetchFilesAndCollections = (ws: WebSocket) => {
  if (!localStorage.getItem("token") && !localStorage.getItem("email") && !localStorage.getItem("password")) {
    localStorage.setItem("token", generateClientToken());
  }
  ws.send(JSON.stringify({
    type: "get_user_files",
    data: {
      token: localStorage.getItem("token") || "",
      email: localStorage.getItem("email") || "",
      password: localStorage.getItem("password") || ""
    }
  }));
  ws.send(JSON.stringify({
    type: "get_user_collections",
    data: {
      token: localStorage.getItem("token") || "",
      email: localStorage.getItem("email") || "",
      password: localStorage.getItem("password") || ""
    }
  }));
}

const getCollection = (id: string, status: Accessor<SocketStatus>, socket: Accessor<WebSocket | undefined>, ctx: AppContextType) => {
    if (ctx.knownCollections()[id]) return;
    if (status() !== "connected") {
        setTimeout(() => getCollection(id, status, socket, ctx), 10);
    } else {
        socket()?.send(JSON.stringify({
            type: "get_collection",
            data: {
                id: id,
                auth: {
                    token: localStorage.getItem("token") || "",
                    email: localStorage.getItem("email") || "",
                    password: localStorage.getItem("password") || ""
                }
            }
        }));
    }
}

export {getFileMD5, formatFileSize, truncateFileName, getFileType, UniversalMessageHandler, generateClientToken, fetchFilesAndCollections, getCollection};